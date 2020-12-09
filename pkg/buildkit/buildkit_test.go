package buildkit

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	bkclient "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/client/llb"
	"github.com/opencontainers/go-digest"

	"github.com/dollarshaveclub/furan/pkg/datalayer"
	"github.com/dollarshaveclub/furan/pkg/generated/furanrpc"
	"github.com/dollarshaveclub/furan/pkg/models"
)

func TestBuildSolver_genSolveOpt(t *testing.T) {
	type args struct {
		b    models.Build
		opts models.BuildOpts
	}
	tests := []struct {
		name    string
		args    args
		want    bkclient.SolveOpt
		wantErr bool
	}{
		{
			name: "build with tags",
			args: args{
				b: models.Build{
					GitHubRepo:   "foo/bar",
					GitHubRef:    "master",
					ImageRepos:   []string{"acme/foo"},
					Tags:         []string{"master", "v1.0.0"},
					CommitSHATag: true,
				},
				opts: models.BuildOpts{
					ContextPath:            "/tmp/asdf",
					CommitSHA:              "zxcvb",
					RelativeDockerfilePath: ".",
					BuildArgs:              map[string]string{"VERSION": "zxcvb"},
				},
			},
			want: bkclient.SolveOpt{
				Exports: []bkclient.ExportEntry{
					bkclient.ExportEntry{
						Type: "image",
						Attrs: map[string]string{
							"push": "true",
							"name": "acme/foo:master,acme/foo:v1.0.0,acme/foo:zxcvb",
						},
					},
				},
				LocalDirs: map[string]string{
					"context":    "/tmp/asdf",
					"dockerfile": ".",
				},
				Frontend: "dockerfile.v0",
				FrontendAttrs: map[string]string{
					"build-arg:VERSION": "zxcvb",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bks := &BuildSolver{}
			got, err := bks.genSolveOpt(tt.args.b, tt.args.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("genSolveOpt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want) {
				t.Errorf("genSolveOpt() got = %v, want %v", got, tt.want)
			}
		})
	}
}

type testBuildKitClient struct {
	MessageInterval    time.Duration
	StatusMessageCount uint
	ReturnError        error
}

func (tbk *testBuildKitClient) Solve(ctx context.Context, def *llb.Definition, opt bkclient.SolveOpt, statusChan chan *bkclient.SolveStatus) (*bkclient.SolveResponse, error) {
	defer close(statusChan)
	for i := uint(0); i < tbk.StatusMessageCount; i++ {
		time.Sleep(tbk.MessageInterval)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled")
		default:
		}
		digest := digest.NewDigestFromBytes(digest.Canonical, []byte("foo"))
		statusChan <- &bkclient.SolveStatus{
			Vertexes: []*bkclient.Vertex{
				&bkclient.Vertex{
					Digest: digest,
					Name:   "foo",
				},
			},
			Statuses: []*bkclient.VertexStatus{
				&bkclient.VertexStatus{
					ID:     "something",
					Vertex: digest,
					Name:   "foo",
				},
			},
			Logs: []*bkclient.VertexLog{
				&bkclient.VertexLog{
					Vertex:    digest,
					Timestamp: time.Now().UTC(),
					Data:      []byte(fmt.Sprintf("this is a build log message: %v", i)),
				},
			},
		}
	}
	return &bkclient.SolveResponse{ExporterResponse: map[string]string{"something": "1234"}}, tbk.ReturnError
}

var _ client = &testBuildKitClient{}

func TestBuildSolver_Build(t *testing.T) {
	type fields struct {
		dl   datalayer.DataLayer
		bc   client
		LogF LogFunc
	}
	type args struct {
		opts models.BuildOpts
	}
	tests := []struct {
		name       string
		fields     fields
		build      models.Build
		args       args
		connecterr bool
		wantEvents uint
		wantErr    bool
		cancel     bool
	}{
		{
			name: "success",
			fields: fields{
				dl: &datalayer.FakeDataLayer{},
				bc: &testBuildKitClient{
					StatusMessageCount: 3,
				},
			},
			build: models.Build{
				GitHubRepo:   "foo/bar",
				GitHubRef:    "master",
				ImageRepos:   []string{"acme/foo"},
				Tags:         []string{"master", "v1.0.0"},
				CommitSHATag: true,
			},
			args: args{
				opts: models.BuildOpts{
					ContextPath:            "/tmp/foo",
					CommitSHA:              "asdf",
					RelativeDockerfilePath: ".",
				},
			},
			wantEvents: 4,
		},
		{
			name: "cancelled",
			fields: fields{
				dl: &datalayer.FakeDataLayer{},
				bc: &testBuildKitClient{
					MessageInterval:    10 * time.Millisecond,
					StatusMessageCount: 3,
				},
			},
			build: models.Build{
				GitHubRepo:   "foo/bar",
				GitHubRef:    "master",
				ImageRepos:   []string{"acme/foo"},
				Tags:         []string{"master", "v1.0.0"},
				CommitSHATag: true,
				Status:       models.BuildStatusRunning,
			},
			args: args{
				opts: models.BuildOpts{
					ContextPath:            "/tmp/foo",
					CommitSHA:              "asdf",
					RelativeDockerfilePath: ".",
				},
			},
			cancel:  true,
			wantErr: true,
		},
		{
			name: "connect error",
			fields: fields{
				dl: &datalayer.FakeDataLayer{},
				bc: &testBuildKitClient{
					StatusMessageCount: 3,
				},
			},
			build: models.Build{
				GitHubRepo:   "foo/bar",
				GitHubRef:    "master",
				ImageRepos:   []string{"acme/foo"},
				Tags:         []string{"master", "v1.0.0"},
				CommitSHATag: true,
			},
			args: args{
				opts: models.BuildOpts{
					ContextPath:            "/tmp/foo",
					CommitSHA:              "asdf",
					RelativeDockerfilePath: ".",
				},
			},
			connecterr: true,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldtimeout := SocketConnectTimeout
			SocketConnectTimeout = 100 * time.Millisecond
			oldretry := SocketConnectRetryDelay
			SocketConnectRetryDelay = 100 * time.Millisecond
			defer func() { SocketConnectTimeout = oldtimeout; SocketConnectRetryDelay = oldretry }()
			var addr string
			eventSink = nil
			listening := make(chan struct{})
			if tt.connecterr {
				addr = "unix:///invalid/path"
				close(listening)
			} else {
				go func() {
					tf, err := ioutil.TempFile("", "furan-test-socket-*")
					if err != nil {
						t.Errorf("error creating temp file: %v", err)
						return
					}
					tf.Close()
					os.Remove(tf.Name())
					addr = "unix://" + tf.Name()
					l, err := net.Listen("unix", tf.Name())
					if err != nil {
						t.Errorf("error listening on socket: %v", err)
						return
					}
					defer l.Close()
					close(listening)
					conn, err := l.Accept()
					if err != nil {
						t.Errorf("error accepting connection: %v", err)
						return
					}
					conn.Close()
				}()
			}
			<-listening
			bks := &BuildSolver{
				dl:   tt.fields.dl,
				bc:   tt.fields.bc,
				addr: addr,
				LogF: tt.fields.LogF,
			}
			ctx, cf := context.WithCancel(context.Background())
			defer cf()
			id, err := tt.fields.dl.CreateBuild(ctx, tt.build)
			if err != nil {
				t.Fatalf("error creating build: %v", err)
			}
			tt.build.ID = id
			tt.args.opts.BuildID = id
			if tt.cancel {
				go func() {
					time.Sleep(20 * time.Millisecond)
					//tt.fields.dl.CancelBuild(ctx, id)
					cf()
				}()
			}
			if err := tt.fields.dl.SetBuildAsRunning(ctx, id); err != nil {
				t.Errorf("error setting build as running: %v", err)
			}
			if err := bks.Build(ctx, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
			}
			time.Sleep(500 * time.Millisecond) // make sure that all events get recorded
			b, err := tt.fields.dl.GetBuildByID(ctx, tt.build.ID)
			if err != nil {
				t.Fatalf("error getting build: %v", err)
			}
			if tt.wantEvents > 0 {
				if len(b.Events) != int(tt.wantEvents) {
					t.Errorf("wanted %v events, got %v: %+v", tt.wantEvents, len(b.Events), b.Events)
				}
			}
		})
	}
}

func Test_imageNames(t *testing.T) {
	type args struct {
		imageRepos []string
		tags       []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "multi repos",
			args: args{
				imageRepos: []string{
					"acme/foo",
					"quay.io/acme/foo",
				},
				tags: []string{
					"master",
					"release",
					"asdf",
				},
			},
			want: []string{
				"acme/foo:master",
				"acme/foo:release",
				"acme/foo:asdf",
				"quay.io/acme/foo:master",
				"quay.io/acme/foo:release",
				"quay.io/acme/foo:asdf",
			},
		},
		{
			name: "single repo",
			args: args{
				imageRepos: []string{
					"acme/foo",
				},
				tags: []string{
					"master",
					"release",
					"asdf",
				},
			},
			want: []string{
				"acme/foo:master",
				"acme/foo:release",
				"acme/foo:asdf",
			},
		},
		{
			name: "no repo",
			args: args{
				imageRepos: []string{},
				tags: []string{
					"master",
					"release",
					"asdf",
				},
			},
			want: []string{},
		},
		{
			name: "no tags",
			args: args{
				imageRepos: []string{
					"acme/foo",
				},
				tags: []string{},
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := imageNames(tt.args.imageRepos, tt.args.tags); !cmp.Equal(got, tt.want) {
				t.Errorf("imageNames() = %v, want %v", got, tt.want)
			}
		})
	}
}

type stubCacheFetcher struct {
	FetchFunc func(ctx context.Context, b models.Build) (string, error)
	GetFunc   func(ctx context.Context, b models.Build, path string) error
}

func (scf *stubCacheFetcher) Fetch(ctx context.Context, b models.Build) (string, error) {
	if scf.FetchFunc != nil {
		return scf.FetchFunc(ctx, b)
	}
	return "", nil
}

func (scf *stubCacheFetcher) Save(ctx context.Context, b models.Build, path string) error {
	if scf.GetFunc != nil {
		return scf.GetFunc(ctx, b, path)
	}
	return nil
}

func TestBuildSolver_loadCache(t *testing.T) {
	type args struct {
		opts models.BuildOpts
	}
	tests := []struct {
		name    string
		args    args
		build   models.Build
		verifyf func(sopt *bkclient.SolveOpt, fetchCalled bool) error
		wantErr bool
	}{
		{
			name: "s3",
			args: args{
				opts: models.BuildOpts{
					Cache: furanrpc.BuildCacheOpts{
						Type: furanrpc.BuildCacheOpts_S3,
					},
				},
			},
			build: models.Build{
				GitHubRepo:   "foo/bar",
				GitHubRef:    "master",
				ImageRepos:   []string{"acme/foo"},
				Tags:         []string{"master", "v1.0.0"},
				CommitSHATag: true,
				Status:       models.BuildStatusRunning,
			},
			verifyf: func(sopt *bkclient.SolveOpt, fetchCalled bool) error {
				if !fetchCalled {
					return fmt.Errorf("expected fetch to be called")
				}
				if i := len(sopt.CacheImports); i != 1 {
					return fmt.Errorf("wanted 1 cache import, got %v", i)
				}
				if t := sopt.CacheImports[0].Type; t != "local" {
					return fmt.Errorf("expected local cache import: %v", t)
				}
				if _, ok := sopt.CacheImports[0].Attrs["src"]; !ok {
					return fmt.Errorf("missing src from cache import attrs")
				}
				if i := len(sopt.CacheExports); i != 1 {
					return fmt.Errorf("wanted 1 cache export, got %v", i)
				}
				if t := sopt.CacheExports[0].Type; t != "local" {
					return fmt.Errorf("expected local cache export: %v", t)
				}
				if _, ok := sopt.CacheExports[0].Attrs["dest"]; !ok {
					return fmt.Errorf("missing dest from cache export attrs")
				}
				mode, ok := sopt.CacheExports[0].Attrs["mode"]
				if !ok {
					return fmt.Errorf("missing mode from cache export attrs")
				}
				if mode != "min" {
					return fmt.Errorf("expected min export mode: %v", mode)
				}
				return nil
			},
		},
		{
			name: "s3 max mode",
			args: args{
				opts: models.BuildOpts{
					Cache: furanrpc.BuildCacheOpts{
						Type:    furanrpc.BuildCacheOpts_S3,
						MaxMode: true,
					},
				},
			},
			build: models.Build{
				GitHubRepo:   "foo/bar",
				GitHubRef:    "master",
				ImageRepos:   []string{"acme/foo"},
				Tags:         []string{"master", "v1.0.0"},
				CommitSHATag: true,
				Status:       models.BuildStatusRunning,
			},
			verifyf: func(sopt *bkclient.SolveOpt, fetchCalled bool) error {
				// other verifications skipped, the test above catches them
				if i := len(sopt.CacheExports); i != 1 {
					return fmt.Errorf("wanted 1 cache export, got %v", i)
				}
				mode, ok := sopt.CacheExports[0].Attrs["mode"]
				if !ok {
					return fmt.Errorf("missing mode from cache export attrs")
				}
				if mode != "max" {
					return fmt.Errorf("expected max export mode: %v", mode)
				}
				return nil
			},
		},
		{
			name: "inline",
			args: args{
				opts: models.BuildOpts{
					Cache: furanrpc.BuildCacheOpts{
						Type: furanrpc.BuildCacheOpts_INLINE,
					},
				},
			},
			build: models.Build{
				GitHubRepo:   "foo/bar",
				GitHubRef:    "master",
				ImageRepos:   []string{"acme/foo"},
				Tags:         []string{"master", "v1.0.0"},
				CommitSHATag: true,
				Status:       models.BuildStatusRunning,
			},
			verifyf: func(sopt *bkclient.SolveOpt, fetchCalled bool) error {
				if fetchCalled {
					return fmt.Errorf("expected fetch to not be called")
				}
				if i := len(sopt.CacheImports); i != 1 {
					return fmt.Errorf("wanted 1 cache import, got %v", i)
				}
				if t := sopt.CacheImports[0].Type; t != "registry" {
					return fmt.Errorf("expected registry cache import: %v", t)
				}
				ref, ok := sopt.CacheImports[0].Attrs["ref"]
				if !ok {
					return fmt.Errorf("missing ref from cache import attrs")
				}
				if ref != "acme/foo" {
					return fmt.Errorf("bad cache import ref: %v", ref)
				}
				if i := len(sopt.CacheExports); i != 1 {
					return fmt.Errorf("wanted 1 cache export, got %v", i)
				}
				if t := sopt.CacheExports[0].Type; t != "inline" {
					return fmt.Errorf("expected inline cache export: %v", t)
				}
				if i := len(sopt.CacheExports[0].Attrs); i != 0 {
					return fmt.Errorf("expected empty cache exports attrs: len: %v", i)
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			dl := &datalayer.FakeDataLayer{}
			id, _ := dl.CreateBuild(ctx, tt.build)
			tt.args.opts.BuildID = id
			var fetchcalled bool
			cf := &stubCacheFetcher{
				FetchFunc: func(ctx context.Context, b models.Build) (string, error) {
					fetchcalled = true
					return "foo", nil
				},
			}
			bks := &BuildSolver{
				dl:   dl,
				s3cf: cf,
				LogF: t.Logf,
			}
			sopts := &bkclient.SolveOpt{}
			cleanup, _, err := bks.loadCache(ctx, tt.args.opts, sopts)
			if cleanup != nil {
				cleanup()
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("loadCache() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.verifyf != nil {
				if err := tt.verifyf(sopts, fetchcalled); err != nil {
					t.Errorf("verify failed: %v", err)
				}
			}
		})
	}
}

func TestBuildSolver_saveCache(t *testing.T) {
	type args struct {
		opts   models.BuildOpts
		expath string
	}
	tests := []struct {
		name      string
		build     models.Build
		args      args
		getfunc   func(ctx context.Context, b models.Build, path string) error
		getcalled bool
		wantErr   bool
	}{
		{
			name: "s3",
			build: models.Build{
				GitHubRepo:   "foo/bar",
				GitHubRef:    "master",
				ImageRepos:   []string{"acme/foo"},
				Tags:         []string{"master", "v1.0.0"},
				CommitSHATag: true,
				Status:       models.BuildStatusRunning,
			},
			args: args{
				opts: models.BuildOpts{
					Cache: furanrpc.BuildCacheOpts{
						Type:    furanrpc.BuildCacheOpts_S3,
						MaxMode: true,
					},
				},
				expath: "/foo/bar",
			},
			getfunc: func(ctx context.Context, b models.Build, path string) error {
				if path != "/foo/bar" {
					return fmt.Errorf("bad path: %v", path)
				}
				return nil
			},
			getcalled: true,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			dl := &datalayer.FakeDataLayer{}
			id, _ := dl.CreateBuild(ctx, tt.build)
			tt.args.opts.BuildID = id
			var getcalled bool
			cf := &stubCacheFetcher{
				GetFunc: func(ctx context.Context, b models.Build, path string) error {
					getcalled = true
					if tt.getfunc != nil {
						return tt.getfunc(ctx, b, path)
					}
					return nil
				},
			}
			bks := &BuildSolver{
				dl:   dl,
				s3cf: cf,
				LogF: t.Logf,
			}
			if err := bks.saveCache(ctx, tt.args.opts, tt.args.expath); (err != nil) != tt.wantErr {
				t.Errorf("saveCache() error = %v, wantErr %v", err, tt.wantErr)
			}
			if getcalled != tt.getcalled {
				t.Errorf("cache fetcher get called flag was %v but wanted %v", getcalled, tt.getcalled)
			}
		})
	}
}
