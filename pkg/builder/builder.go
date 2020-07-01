package builder

import (
	"context"

	"github.com/gofrs/uuid"

	"github.com/dollarshaveclub/furan/pkg/datalayer"
	"github.com/dollarshaveclub/furan/pkg/models"
)

type BuildRunner interface {
	Build(ctx context.Context, opts models.BuildOpts) error
}

type JobRunner interface {
	Run(build models.Build) (models.Job, error)
}

// Manager is an object that performs high-level management of image builds
type Manager struct {
	BRunner BuildRunner
	JRunner JobRunner
	DL      datalayer.DataLayer
}

// Start starts a single build using JobRunner and returns immediately.
func (m *Manager) Start(ctx context.Context, opts models.BuildOpts) (models.Job, error) {
	return nil, nil
}

// Run synchronously executes a build using BuildRunner
func (m *Manager) Run(ctx context.Context, opts models.BuildOpts) error {
	return nil
}

// Monitor monitors a build that's currently running, sending status messages on msgs until the build finishes
// If the build fails, a non-nil error is returned
func (m *Manager) Monitor(ctx context.Context, buildid uuid.UUID, msgs chan string) error {
	return nil
}

// Cancel aborts a build that's currently running
func (m *Manager) Cancel(buildid uuid.UUID) error {
	return nil
}
