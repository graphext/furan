package builder

import (
	"context"
	"fmt"

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

// Manager is an object that performs high-level management of image builds
type Manager struct {
	BRunner BuildRunner
	JRunner JobRunner
	DL      datalayer.DataLayer
}

func (m *Manager) validateOpts(opts models.BuildOpts, wantStatus models.BuildStatus) error {
	return nil
}

// Start starts a single build using JobRunner and waits for the build to begin running, or returns error
func (m *Manager) Start(ctx context.Context, opts models.BuildOpts) error {
	if err := m.validateOpts(opts, models.BuildStatusNotStarted); err != nil {
		return fmt.Errorf("invalid build opts: %w", err)
	}
	if m.JRunner == nil {
		return fmt.Errorf("job runner is nil")
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
	if err := m.validateOpts(opts, models.BuildStatusNotStarted); err != nil {
		return fmt.Errorf("invalid build opts: %w", err)
	}
	if m.BRunner == nil {
		return fmt.Errorf("build runner is nil")
	}
	b, err := m.DL.GetBuildByID(ctx, opts.BuildID)
	if err != nil {
		return fmt.Errorf("error getting build: %w", err)
	}

	// TODO: check if tags exist (if requested)

	if err := m.DL.SetBuildAsRunning(ctx, b.ID); err != nil {
		return fmt.Errorf("error setting build status to running: %w", err)
	}
	err = m.BRunner.Build(ctx, opts)
	status := models.BuildStatusSuccess
	if err != nil {
		status = models.BuildStatusFailure
	}
	if err2 := m.DL.SetBuildAsCompleted(ctx, b.ID, status); err2 != nil {
		err = multierror.Append(err, fmt.Errorf("error setting build as completed: %w", err2))
	}
	return err
}

// Monitor monitors a build that's currently running, sending status messages on msgs until the build finishes
// If the build fails, a non-nil error is returned
func (m *Manager) Monitor(ctx context.Context, buildid uuid.UUID, msgs chan string) error {
	if m.DL == nil {
		return fmt.Errorf("datalayer is nil")
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
	if m.DL == nil {
		return fmt.Errorf("datalayer is nil")
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
