package buildkit

import (
	"context"

	"github.com/dollarshaveclub/furan/pkg/models"
)

type FakeBuilder struct {
	BuildFunc func(ctx context.Context, opts models.BuildOpts) error
}

func (fr *FakeBuilder) Build(ctx context.Context, opts models.BuildOpts) error {
	if fr.BuildFunc != nil {
		return fr.BuildFunc(ctx, opts)
	}
	return nil
}

var _ models.Builder = &FakeBuilder{}
