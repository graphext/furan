package github

import "context"

type FakeFetcher struct {
	FetchFunc        func(ctx context.Context, repo string, ref string, destinationPath string) error
	GetCommitSHAFunc func(ctx context.Context, repo string, ref string) (string, error)
}

func (ff *FakeFetcher) Fetch(ctx context.Context, repo string, ref string, destinationPath string) error {
	if ff.FetchFunc != nil {
		return ff.FetchFunc(ctx, repo, ref, destinationPath)
	}
	return nil
}

func (ff *FakeFetcher) GetCommitSHA(ctx context.Context, repo string, ref string) (string, error) {
	if ff.GetCommitSHAFunc != nil {
		return ff.GetCommitSHAFunc(ctx, repo, ref)
	}
	return "", nil
}
