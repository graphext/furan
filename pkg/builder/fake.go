package builder

import (
	"context"

	"github.com/dollarshaveclub/furan/pkg/models"
)

type FakeBuildManager struct {
	StartFunc func(ctx context.Context, opts models.BuildOpts) error
	RunFunc   func(ctx context.Context, opts models.BuildOpts) error
}

var _ models.BuildManager = &FakeBuildManager{}

func (fb *FakeBuildManager) Start(ctx context.Context, opts models.BuildOpts) error {
	if fb.StartFunc != nil {
		return fb.StartFunc(ctx, opts)
	}
	return nil
}
func (fb *FakeBuildManager) Run(ctx context.Context, opts models.BuildOpts) error {
	if fb.RunFunc != nil {
		return fb.RunFunc(ctx, opts)
	}
	return nil
}
