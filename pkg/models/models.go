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
	BuildStatusBuilding
	BuildStatusPushing
	BuildStatusSuccess
	BuildStatusBuildFailure
	BuildStatusPushFailure
	BuildStatusNotNeeded
)

type Build struct {
	ID                          uuid.UUID
	Created, Updated, Completed time.Time
	GitHubRepo, GitHubRef       string
	ImageRepo                   string
	Tags                        []string
	CommitSHATag                bool
	Request                     furanrpc.BuildRequest
	Status                      BuildStatus
	Events                      []string
}

func (b Build) CanAddEvent() bool {
	switch b.Status {
	case BuildStatusUnknown:
		fallthrough
	case BuildStatusNotStarted:
		fallthrough
	case BuildStatusSuccess:
		fallthrough
	case BuildStatusBuildFailure:
		fallthrough
	case BuildStatusPushFailure:
		fallthrough
	case BuildStatusNotNeeded:
		return false
	}
	return true
}

func (b Build) Running() bool {
	return b.CanAddEvent()
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
	// Done returns a signal that the Job has completed successfully
	Done() chan struct{}
	// Lobs returns all pod logs associated with the Job
	Logs() (map[string]map[string][]byte, error)
}
