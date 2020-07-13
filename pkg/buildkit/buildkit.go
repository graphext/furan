package buildkit

import (
	"context"
	"strings"

	"github.com/moby/buildkit/client/llb"

	"github.com/dollarshaveclub/furan/pkg/datalayer"
	"github.com/dollarshaveclub/furan/pkg/models"

	"fmt"

	bkclient "github.com/moby/buildkit/client"
)

type client interface {
	Solve(ctx context.Context, def *llb.Definition, opt bkclient.SolveOpt, statusChan chan *bkclient.SolveStatus) (*bkclient.SolveResponse, error)
}

var _ client = &bkclient.Client{}

type LogFunc func(msg string, args ...interface{})

type BuildSolver struct {
	dl   datalayer.DataLayer
	bc   client
	LogF LogFunc
}

func (bks *BuildSolver) log(msg string, args ...interface{}) {
	if bks.LogF != nil {
		bks.LogF(msg, args...)
	}
}

func NewBuildSolver(addr string, dl datalayer.DataLayer) (*BuildSolver, error) {
	if !strings.HasPrefix(addr, "unix://") {
		return nil, fmt.Errorf("addr must be a unix socket")
	}
	bc, err := bkclient.New(context.Background(), addr, bkclient.WithFailFast())
	if err != nil {
		return nil, fmt.Errorf("error getting buildkit client: %w", err)
	}
	return &BuildSolver{
		dl: dl,
		bc: bc,
	}, nil
}

func imageNames(imageRepos []string, tags []string) []string {
	out := make([]string, len(imageRepos)*len(tags))
	for i := range imageRepos {
		for j := range tags {
			out[(i*len(tags))+j] = imageRepos[i] + ":" + tags[j]
		}
	}
	return out
}

func (bks *BuildSolver) genSolveOpt(b models.Build, opts models.BuildOpts) (bkclient.SolveOpt, error) {
	tags := b.Tags
	if b.CommitSHATag {
		tags = append(tags, opts.CommitSHA)
	}
	export := bkclient.ExportEntry{
		Type: "image",
		Attrs: map[string]string{
			"push": "true",
			"name": strings.Join(imageNames(b.ImageRepos, tags), ","),
		},
	}
	sopts := bkclient.SolveOpt{
		Exports:       []bkclient.ExportEntry{export},
		Frontend:      "dockerfile.v0",
		FrontendAttrs: map[string]string{},
		LocalDirs: map[string]string{
			"context":    opts.ContextPath,
			"dockerfile": opts.RelativeDockerfilePath,
		},
	}
	for k, v := range opts.BuildArgs {
		sopts.FrontendAttrs["build-arg:"+k] = v
	}
	if opts.CacheImportPath != "" {
		sopts.CacheImports = []bkclient.CacheOptionsEntry{
			bkclient.CacheOptionsEntry{
				Type: "local",
				Attrs: map[string]string{
					"src":  opts.CacheImportPath,
					"mode": "max",
				},
			},
		}
	}
	if opts.CacheExportPath != "" {
		sopts.CacheExports = []bkclient.CacheOptionsEntry{
			bkclient.CacheOptionsEntry{
				Type: "local",
				Attrs: map[string]string{
					"dest": opts.CacheExportPath,
					"mode": "max",
				},
			},
		}
	}
	return sopts, nil
}

// Build performs the build defined by opts
func (bks *BuildSolver) Build(ctx context.Context, opts models.BuildOpts) error {
	b, err := bks.dl.GetBuildByID(ctx, opts.BuildID)
	if err != nil {
		return fmt.Errorf("error getting build: %v: %w", opts.BuildID, err)
	}
	sopts, err := bks.genSolveOpt(b, opts)
	if err != nil {
		return fmt.Errorf("error generating solve options: %w", err)
	}
	c := make(chan *bkclient.SolveStatus)
	defer close(c)
	ctx2, cf := context.WithCancel(ctx)
	defer cf()
	cxl := make(chan struct{})
	go func() {
		err := bks.dl.ListenForCancellation(ctx2, b.ID)
		if err != nil {
			bks.log("error listening for cancellation: %v", err)
			return
		}
		close(cxl)
	}()
	go func() {
		select {
		case <-cxl:
			if err := bks.dl.AddEvent(ctx2, b.ID, "build cancellation request received"); err != nil {
				bks.log("error adding cancellation event: build: %v: %v", opts.BuildID, err)
			}
			cf()
		case <-ctx2.Done():
		}
	}()
	go func() {
		for {
			select {
			case <-ctx2.Done():
				return
			case ss := <-c:
				if ss == nil { // channel closed
					return
				}
				for _, l := range ss.Logs {
					if l != nil {
						if err := bks.dl.AddEvent(ctx2, opts.BuildID, string(l.Data)); err != nil {
							bks.log("error adding event: build: %v: %v", opts.BuildID, err)
						}
					}
				}
			}
		}
	}()
	resp, err := bks.bc.Solve(ctx2, nil, sopts, c)
	if err != nil {
		return fmt.Errorf("error running solver: %w", err)
	}
	msg := fmt.Sprintf("solve success: %+v", resp.ExporterResponse)
	if err := bks.dl.AddEvent(ctx2, opts.BuildID, msg); err != nil {
		bks.log("error adding success event: build: %v: %v", opts.BuildID, err)
	}
	return nil
}
