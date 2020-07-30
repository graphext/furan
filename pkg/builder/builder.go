package builder

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gofrs/uuid"
	"github.com/hashicorp/go-multierror"

	"github.com/dollarshaveclub/furan/pkg/datalayer"
	"github.com/dollarshaveclub/furan/pkg/models"
)

// BuildRunner describes an object that can perform a build synchronously
type BuildRunner interface {
	Build(ctx context.Context, opts models.BuildOpts) error
}

// JobRunner describes an object that can asynchronously run a job that executes a build
type JobRunner interface {
	Run(build models.Build) (models.Job, error)
}

type TagChecker interface {
	AllTagsExist(tags []string, repo string) (bool, []string, error)
}

// Manager is an object that performs high-level management of image builds
type Manager struct {
	BRunner        BuildRunner
	JRunner        JobRunner
	TCheck         TagChecker
	FetcherFactory func(token string) models.CodeFetcher
	GitHubTokenKey [32]byte
	DL             datalayer.DataLayer
}

func (m *Manager) validateDeps() bool {
	return m != nil && m.BRunner != nil && m.JRunner != nil && m.TCheck != nil && m.DL != nil && m.FetcherFactory != nil
}

// Start starts a single build using JobRunner and waits for the build to begin running, or returns error
func (m *Manager) Start(ctx context.Context, opts models.BuildOpts) error {
	if !m.validateDeps() {
		return fmt.Errorf("missing dependency: %#v", m)
	}
	b, err := m.DL.GetBuildByID(ctx, opts.BuildID)
	if err != nil {
		return fmt.Errorf("error getting build: %w", err)
	}
	j, err := m.JRunner.Run(b)
	if err != nil {
		return fmt.Errorf("error running build job: %w", err)
	}
	defer j.Close()

	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled")
	case err := <-j.Error():
		return fmt.Errorf("job error: %w", err)
	case <-j.Running():
		return nil
	}
}

// Run synchronously executes a build using BuildRunner
func (m *Manager) Run(ctx context.Context, opts models.BuildOpts) error {
	if !m.validateDeps() {
		return fmt.Errorf("missing dependency: %#v", m)
	}
	b, err := m.DL.GetBuildByID(ctx, opts.BuildID)
	if err != nil {
		return fmt.Errorf("error getting build: %w", err)
	}

	tkn, err := b.GetGitHubCredential(m.GitHubTokenKey)
	if err != nil {
		return fmt.Errorf("error decrypting github token: %w", err)
	}

	fetcher := m.FetcherFactory(tkn)

	csha, err := fetcher.GetCommitSHA(ctx, b.GitHubRepo, b.GitHubRef)
	if err != nil {
		return fmt.Errorf("error getting commit sha: %w", err)
	}

	opts.CommitSHA = csha

	if b.Request.SkipIfExists {
		allexist := true
		for _, irepo := range b.ImageRepos {
			exists, _, err := m.TCheck.AllTagsExist(append(b.Tags, opts.CommitSHA), irepo)
			if err != nil {
				return fmt.Errorf("error checking for tags: %v: %w", irepo, err)
			}
			allexist = allexist && exists
			if !allexist {
				break
			}
		}
		if allexist {
			// set build as skipped
			if err := m.DL.SetBuildAsCompleted(ctx, b.ID, models.BuildStatusSkipped); err != nil {
				return fmt.Errorf("error setting build as skipped: %w", err)
			}
			return nil
		}
	}

	tdir, err := ioutil.TempDir("", "furan-build-context-*")
	if err != nil {
		return fmt.Errorf("error creating temp dir for build context: %w", err)
	}
	defer os.RemoveAll(tdir)
	err = fetcher.Fetch(ctx, b.GitHubRepo, b.GitHubRef, tdir)
	if err != nil {
		return fmt.Errorf("error fetching repo: %w", err)
	}

	opts.ContextPath = tdir

	if err := m.DL.SetBuildAsRunning(ctx, b.ID); err != nil {
		return fmt.Errorf("error setting build status to running: %w", err)
	}

	ctx2, cf := context.WithCancel(ctx)
	defer cf()

	go func() {
		if err := m.DL.ListenForCancellation(ctx2, b.ID); err == nil {
			cf()
		}
	}()

	err = m.BRunner.Build(ctx2, opts)
	status := models.BuildStatusSuccess
	if err != nil {
		status = models.BuildStatusFailure
		select {
		case <-ctx2.Done():
			status = models.BuildStatusCancelled
		default:
			break
		}
	}
	if err2 := m.DL.SetBuildAsCompleted(ctx2, b.ID, status); err2 != nil {
		err = multierror.Append(err, fmt.Errorf("error setting build as completed: %w", err2))
	}
	return err
}

// Monitor monitors a build that's currently running, sending status messages on msgs until the build finishes
func (m *Manager) Monitor(ctx context.Context, buildid uuid.UUID, msgs chan string) error {
	if !m.validateDeps() {
		return fmt.Errorf("missing dependency: %#v", m)
	}
	ctx, cf := context.WithCancel(ctx)
	defer cf()
	go m.DL.ListenForBuildEvents(ctx, buildid, msgs)
	bs, err := m.DL.ListenForBuildCompleted(ctx, buildid)
	if err != nil {
		return fmt.Errorf("error listening for build completion: %w", err)
	}
	msgs <- fmt.Sprintf("build completed with status %v", bs)
	return nil
}

// Cancel aborts a build that's currently running, optionally waiting for build to signal that it has aborted
func (m *Manager) Cancel(ctx context.Context, buildid uuid.UUID, wait bool) error {
	if !m.validateDeps() {
		return fmt.Errorf("missing dependency: %#v", m)
	}
	if err := m.DL.CancelBuild(ctx, buildid); err != nil {
		return fmt.Errorf("error cancelling build: %v", err)
	}
	if wait {
		_, err := m.DL.ListenForBuildCompleted(ctx, buildid)
		return err
	}
	return nil
}
