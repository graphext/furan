package buildkit

import (
	"context"
	"io/ioutil"
	"os"
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
	s3cf models.CacheFetcher
	bc   client
	LogF LogFunc
}

func (bks *BuildSolver) log(msg string, args ...interface{}) {
	if bks.LogF != nil {
		bks.LogF(msg, args...)
	}
}

func NewBuildSolver(addr string, s3cf models.CacheFetcher, dl datalayer.DataLayer) (*BuildSolver, error) {
	if !strings.HasPrefix(addr, "unix://") {
		return nil, fmt.Errorf("addr must be a unix socket")
	}
	bc, err := bkclient.New(context.Background(), addr, bkclient.WithFailFast())
	if err != nil {
		return nil, fmt.Errorf("error getting buildkit client: %w", err)
	}
	return &BuildSolver{
		dl:   dl,
		s3cf: s3cf,
		bc:   bc,
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

// loadCache fetches and configures this build cache (if requested & available) and sets sopts appropriately
// A cleanup function is returned
func (bks *BuildSolver) loadCache(ctx context.Context, opts models.BuildOpts, sopts *bkclient.SolveOpt) (func(), error) {
	b, err := bks.dl.GetBuildByID(ctx, opts.BuildID)
	if err != nil {
		return func() {}, fmt.Errorf("error getting build by id: %w", err)
	}

	switch opts.Cache.Type {
	case models.S3CacheType:
		cleanup := func() {}
		path, err := bks.s3cf.Fetch(b)
		if err != nil {
			return cleanup, fmt.Errorf("error fetching cache from s3: %w", err)
		}
		cleanup = func() { os.RemoveAll(path) }
		sopts.CacheImports = []bkclient.CacheOptionsEntry{
			bkclient.CacheOptionsEntry{
				Type: "local",
				Attrs: map[string]string{
					"src": path,
				},
			},
		}
		exportpath, err := ioutil.TempDir("", "build-cache-export-*")
		if err != nil {
			return cleanup, fmt.Errorf("error creating cache export path: %w", err)
		}
		cleanup = func() { os.RemoveAll(path); os.RemoveAll(exportpath) }
		mode := "min"
		if opts.Cache.MaxMode {
			mode = "max"
		}
		sopts.CacheExports = []bkclient.CacheOptionsEntry{
			bkclient.CacheOptionsEntry{
				Type: "local",
				Attrs: map[string]string{
					"dest": exportpath,
					"mode": mode,
				},
			},
		}
		return cleanup, nil
	case models.InlineCacheType:
		sopts.CacheImports = make([]bkclient.CacheOptionsEntry, len(b.ImageRepos))
		for i := range b.ImageRepos {
			sopts.CacheImports[i] = bkclient.CacheOptionsEntry{
				Type: "registry",
				Attrs: map[string]string{
					"ref": b.ImageRepos[i],
				},
			}
		}
		sopts.CacheExports = []bkclient.CacheOptionsEntry{
			bkclient.CacheOptionsEntry{
				Type: "inline",
				// inline exporter supports min mode only
			},
		}
	}

	return func() {}, nil
}

// saveCache persists the exported build cache (if requested & available)
func (bks *BuildSolver) saveCache(ctx context.Context, opts models.BuildOpts, sopts bkclient.SolveOpt) error {
	b, err := bks.dl.GetBuildByID(ctx, opts.BuildID)
	if err != nil {
		return fmt.Errorf("error getting build by id: %w", err)
	}

	// we only need to explicitly persist cache if using S3 (w/ local exporter)
	if opts.Cache.Type == models.S3CacheType {
		if i := len(sopts.CacheExports); i != 1 {
			return fmt.Errorf("unexpected CacheExports length (wanted 1): %v", i)
		}
		if t := sopts.CacheExports[0].Type; t != "local" {
			return fmt.Errorf("unexpected CacheExports type (wanted local): %v", t)
		}
		expath, ok := sopts.CacheExports[0].Attrs["dest"]
		if !ok {
			return fmt.Errorf("cache exports dest attribute is missing: %#v", sopts.CacheExports[0])
		}
		if err := bks.s3cf.Save(b, expath); err != nil {
			return fmt.Errorf("error saving exported cache to S3: %w", err)
		}
	}

	return nil
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
	cleanup, err := bks.loadCache(ctx, opts, &sopts)
	if err != nil {
		// non-fatal error if we can't load cache for some reason
		bks.dl.AddEvent(ctx, b.ID, fmt.Sprintf("warning: error loading cache: %v", err))
	}
	defer cleanup()

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
	if err := bks.saveCache(ctx, opts, sopts); err != nil {
		// non-fatal error if we can't save cache
		bks.dl.AddEvent(ctx2, opts.BuildID, fmt.Sprintf("warning: error saving build cache: %v", err))
	}
	msg := fmt.Sprintf("solve success: %+v", resp.ExporterResponse)
	if err := bks.dl.AddEvent(ctx2, opts.BuildID, msg); err != nil {
		bks.log("error adding success event: build: %v: %v", opts.BuildID, err)
	}
	return nil
}
