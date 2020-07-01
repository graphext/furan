package buildkit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	bkclient "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/client/llb"

	"github.com/dollarshaveclub/furan/pkg/datalayer"
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
					ImageRepo:    "acme/foo",
					Tags:         []string{"master", "v1.0.0"},
					CommitSHATag: true,
				},
				opts: models.BuildOpts{
					ContextPath:            "/tmp/asdf",
					CommitSHA:              "zxcvb",
					RelativeDockerfilePath: ".",
					BuildArgs:              map[string]string{"VERSION": "zxcvb"},
					CacheImportPath:        "/tmp/cache/input",
					CacheExportPath:        "/tmp/cache/output",
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
				CacheExports: []bkclient.CacheOptionsEntry{
					bkclient.CacheOptionsEntry{
						Type: "local",
						Attrs: map[string]string{
							"dest": "/tmp/cache/output",
							"mode": "max",
						},
					},
				},
				CacheImports: []bkclient.CacheOptionsEntry{
					bkclient.CacheOptionsEntry{
						Type: "local",
						Attrs: map[string]string{
							"src":  "/tmp/cache/input",
							"mode": "max",
						},
					},
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
	for i := uint(0); i < tbk.StatusMessageCount; i++ {
		time.Sleep(tbk.MessageInterval)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled")
		default:
		}
		statusChan <- &bkclient.SolveStatus{
			Logs: []*bkclient.VertexLog{
				&bkclient.VertexLog{
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
				ImageRepo:    "acme/foo",
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
				ImageRepo:    "acme/foo",
				Tags:         []string{"master", "v1.0.0"},
				CommitSHATag: true,
				Status:       models.BuildStatusBuilding,
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bks := &BuildSolver{
				dl:   tt.fields.dl,
				bc:   tt.fields.bc,
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
					tt.fields.dl.CancelBuild(ctx, id)
				}()
			}
			if err := bks.Build(ctx, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
			}
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
