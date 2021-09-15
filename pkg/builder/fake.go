package builder

import (
	"context"

	"github.com/gofrs/uuid"

	"github.com/dollarshaveclub/furan/v2/pkg/models"
)

type FakeBuildManager struct {
	StartFunc func(ctx context.Context, opts models.BuildOpts) error
	RunFunc   func(ctx context.Context, id uuid.UUID) error
}

var _ models.BuildManager = &FakeBuildManager{}

func (fb *FakeBuildManager) Start(ctx context.Context, opts models.BuildOpts) error {
	if fb.StartFunc != nil {
		return fb.StartFunc(ctx, opts)
	}
	return nil
}
func (fb *FakeBuildManager) Run(ctx context.Context, id uuid.UUID) error {
	if fb.RunFunc != nil {
		return fb.RunFunc(ctx, id)
	}
	return nil
}
