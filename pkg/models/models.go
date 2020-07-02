package models

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/dollarshaveclub/furan/pkg/generated/furanrpc"
)

//go:generate stringer -type=BuildStatus

type BuildStatus int

const (
	BuildStatusUnknown BuildStatus = iota
	BuildStatusNotStarted
	BuildStatusSkipped
	BuildStatusRunning
	BuildStatusFailure
	BuildStatusSuccess
	BuildStatusCancelled
)

type Build struct {
	ID                          uuid.UUID
	Created, Updated, Completed time.Time
	GitHubRepo, GitHubRef       string
	ImageRepos                  []string
	Tags                        []string
	CommitSHATag                bool
	DisableBuildCache           bool
	Request                     furanrpc.BuildRequest
	Status                      BuildStatus
	Events                      []string
}

func (b Build) CanAddEvent() bool {
	return b.Running()
}

func (b Build) Running() bool {
	return b.Status == BuildStatusRunning
}

// BuildOpts models all options required to perform a build
type BuildOpts struct {
	BuildID                          uuid.UUID
	ContextPath, CommitSHA           string
	RelativeDockerfilePath           string
	BuildArgs                        map[string]string
	CacheImportPath, CacheExportPath string
}

// Job describes methods on a single abstract build job
type Job interface {
	// Close cleans up any resources associated with this Job
	Close()
	// Error returns a channel that will contain any errors associated with this Job
	Error() chan error
	// Done returns a channel that signals that the Job has completed successfully
	Done() chan struct{}
	// Logs returns all pod logs associated with the Job
	Logs() (map[string]map[string][]byte, error)
}
