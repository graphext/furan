package grpc

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/dollarshaveclub/furan/pkg/builder"
	"github.com/dollarshaveclub/furan/pkg/datalayer"
	"github.com/dollarshaveclub/furan/pkg/generated/furanrpc"
	"github.com/dollarshaveclub/furan/pkg/github"
	"github.com/dollarshaveclub/furan/pkg/models"
)

func TestServer_StartBuild(t *testing.T) {
	type fields struct {
		Opts Options
	}
	type args struct {
		req *furanrpc.BuildRequest
	}
	flds := fields{Opts: Options{
		TraceSvcName: "",
		Cache:        models.CacheOpts{},
		LogFunc:      nil,
	}}
	req := &furanrpc.BuildRequest{
		Build: &furanrpc.BuildDefinition{
			GithubRepo:       "acme/foo",
			GithubCredential: "asdf1234",
			Ref:              "master",
			Tags:             []string{"master", "v1.0"},
			TagWithCommitSha: true,
		},
		Push: &furanrpc.PushDefinition{
			Registries: []*furanrpc.PushRegistryDefinition{
				&furanrpc.PushRegistryDefinition{
					Repo: "quay.io/acme/foo",
				},
			},
		},
		SkipIfExists: true,
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		builderr  error
		getshaerr error
		rundelay  time.Duration
		wantErr   bool
	}{
		{
			name:   "success",
			fields: flds,
			args: args{
				req: req,
			},
			rundelay: 10 * time.Millisecond,
		},
		{
			name:   "build start error",
			fields: flds,
			args: args{
				req: req,
			},
			rundelay: 10 * time.Millisecond,
			builderr: fmt.Errorf("error starting build"),
		},
		{
			name: "job handoff timeout",
			fields: fields{
				Opts: Options{
					JobHandoffTimeout: 10 * time.Millisecond,
				},
			},
			args: args{
				req: req,
			},
			rundelay: 100 * time.Second,
		},
		{
			name:   "invalid req: empty github repo",
			fields: flds,
			args: args{
				req: &furanrpc.BuildRequest{
					Build: &furanrpc.BuildDefinition{
						GithubRepo:       "",
						GithubCredential: "asdf1234",
						Ref:              "master",
						Tags:             []string{"master", "v1.0"},
						TagWithCommitSha: true,
					},
					Push: &furanrpc.PushDefinition{
						Registries: []*furanrpc.PushRegistryDefinition{
							&furanrpc.PushRegistryDefinition{
								Repo: "quay.io/acme/foo",
							},
						},
					},
					SkipIfExists: true,
				},
			},
			rundelay: 10 * time.Millisecond,
			wantErr:  true,
		},
		{
			name:   "invalid req: empty ref",
			fields: flds,
			args: args{
				req: &furanrpc.BuildRequest{
					Build: &furanrpc.BuildDefinition{
						GithubRepo:       "acme/foo",
						GithubCredential: "asdf1234",
						Ref:              "",
						Tags:             []string{"master", "v1.0"},
						TagWithCommitSha: true,
					},
					Push: &furanrpc.PushDefinition{
						Registries: []*furanrpc.PushRegistryDefinition{
							&furanrpc.PushRegistryDefinition{
								Repo: "quay.io/acme/foo",
							},
						},
					},
					SkipIfExists: true,
				},
			},
			rundelay: 10 * time.Millisecond,
			wantErr:  true,
		},
		{
			name:   "invalid req: no image repos",
			fields: flds,
			args: args{
				req: &furanrpc.BuildRequest{
					Build: &furanrpc.BuildDefinition{
						GithubRepo:       "acme/foo",
						GithubCredential: "asdf1234",
						Ref:              "master",
						Tags:             []string{"master", "v1.0"},
						TagWithCommitSha: true,
					},
					Push: &furanrpc.PushDefinition{
						Registries: []*furanrpc.PushRegistryDefinition{},
					},
					SkipIfExists: true,
				},
			},
			rundelay: 10 * time.Millisecond,
			wantErr:  true,
		},
		{
			name:   "invalid req: invalid commit sha",
			fields: flds,
			args: args{
				req: &furanrpc.BuildRequest{
					Build: &furanrpc.BuildDefinition{
						GithubRepo:       "acme/foo",
						GithubCredential: "asdf1234",
						Ref:              "master",
						Tags:             []string{"master", "v1.0"},
						TagWithCommitSha: true,
					},
					Push: &furanrpc.PushDefinition{
						Registries: []*furanrpc.PushRegistryDefinition{
							&furanrpc.PushRegistryDefinition{
								Repo: "quay.io/acme/foo",
							},
						},
					},
					SkipIfExists: true,
				},
			},
			rundelay:  10 * time.Millisecond,
			getshaerr: fmt.Errorf("bad commit sha"),
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			dl := &datalayer.FakeDataLayer{}
			bm := &builder.FakeBuildManager{
				StartFunc: func(ctx context.Context, opts models.BuildOpts) error {
					df := "Dockerfile"
					if tt.args.req.Build.DockerfilePath != "" {
						df = tt.args.req.Build.DockerfilePath
					}
					if opts.RelativeDockerfilePath != df {
						t.Errorf("Start: bad dockerfile path: %v (wanted %v)", opts.RelativeDockerfilePath, df)
					}
					if !tt.args.req.Build.DisableBuildCache {
						if !cmp.Equal(DefaultCacheOpts, opts.Cache) {
							t.Errorf("Start: bad cache opts: %#v (wanted %#v)", opts.Cache, DefaultCacheOpts)
						}
					}
					if !cmp.Equal(tt.args.req.Build.Args, opts.BuildArgs) {
						t.Errorf("Start: bad build args: %v (wanted %v)", opts.BuildArgs, tt.args.req.Build.Args)
					}
					if tt.builderr == nil {
						go func() {
							time.Sleep(tt.rundelay) // give time for listener
							if err := dl.SetBuildAsRunning(ctx, opts.BuildID); err != nil {
								t.Errorf("error setting build as running: %v", err)
							}
						}()
					}
					return tt.builderr
				}}
			key := make([]byte, 32)
			rand.Read(key)
			copy(tt.fields.Opts.CredentialDecryptionKey[:], key)
			tt.fields.Opts.LogFunc = t.Logf
			gr := &Server{
				DL: dl,
				BM: bm,
				CFFactory: func(token string) models.CodeFetcher {
					if token != tt.args.req.Build.GithubCredential {
						t.Errorf("bad credential: %v (wanted %v)", token, tt.args.req.Build.GithubCredential)
					}
					return &github.FakeFetcher{
						GetCommitSHAFunc: func(ctx context.Context, repo string, ref string) (string, error) {
							if repo != tt.args.req.Build.GithubRepo {
								t.Errorf("get commit sha: bad repo %v", repo)
							}
							if ref != tt.args.req.Build.Ref {
								t.Errorf("get commit sha: bad ref %v", ref)
							}
							return "asdf", tt.getshaerr
						},
					}
				},
				Opts: tt.fields.Opts,
			}
			got, err := gr.StartBuild(ctx, tt.args.req)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("StartBuild() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			id := uuid.Must(uuid.FromString(got.BuildId))
			b, err := dl.GetBuildByID(ctx, id)
			if err != nil {
				t.Errorf("error validating build id: %v", err)
			}
			// make sure cleartext credential is redacted, and encrypted token matches
			if b.Request.Build.GithubCredential != "" {
				t.Errorf("expected empty request credential")
			}
			tkn, err := b.GetGitHubCredential(tt.fields.Opts.CredentialDecryptionKey)
			if err != nil {
				t.Errorf("error decrypting credential: %v", err)
			}
			if tkn != tt.args.req.Build.GithubCredential {
				t.Errorf("bad credential: %v (wanted %v)", tkn, tt.args.req.Build.GithubCredential)
			}
			if i := len(b.ImageRepos); i != len(tt.args.req.Push.Registries) {
				t.Errorf("bad number of image repos: %v (wanted %v)", len(tt.args.req.Push.Registries), i)
			}
			if !cmp.Equal(b.Tags, tt.args.req.Build.Tags) {
				t.Errorf("bad tags: %v (wanted %v)", b.Tags, tt.args.req.Build.Tags)
			}
			if b.CommitSHATag != tt.args.req.Build.TagWithCommitSha {
				t.Errorf("bad CommitSHATag: %v (wanted %v)", b.CommitSHATag, tt.args.req.Build.TagWithCommitSha)
			}
			if b.DisableBuildCache != tt.args.req.Build.DisableBuildCache {
				t.Errorf("bad DisableBuildCache: %v (wanted %v)", b.DisableBuildCache, tt.args.req.Build.DisableBuildCache)
			}
			ctx, cf := context.WithTimeout(ctx, 2*tt.rundelay)
			defer cf()

			if tt.builderr == nil && tt.fields.Opts.JobHandoffTimeout == 0 {
				if err := dl.ListenForBuildRunning(ctx, id); err != nil {
					t.Errorf("error listening for build running: %v", err)
				}
			} else {
				bs, err := dl.ListenForBuildCompleted(ctx, id)
				if err != nil {
					t.Errorf("error listening for build running: %v", err)
				}
				// build start error, so build status should be failure
				if bs != models.BuildStatusFailure {
					t.Errorf("bad status (wanted Failure): %v", bs)
				}
			}

		})
	}
}

func TestServer_GetBuildStatus(t *testing.T) {
	tests := []struct {
		name        string
		b           *models.Build
		req         *furanrpc.BuildStatusRequest
		wantErr     bool
		wantErrCode codes.Code
	}{
		{
			name: "success",
			b: &models.Build{
				GitHubRepo: "acme/foo",
				GitHubRef:  "master",
				ImageRepos: []string{"quay.io/acme/foo"},
				Status:     models.BuildStatusNotStarted,
			},
		},
		{
			name:        "not found",
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "bad uuid",
			req: &furanrpc.BuildStatusRequest{
				BuildId: "baduuid",
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			dl := &datalayer.FakeDataLayer{}
			var id uuid.UUID
			if tt.b == nil {
				id = uuid.Must(uuid.NewV4())
			} else {
				id, _ = dl.CreateBuild(ctx, *tt.b)
			}
			gr := &Server{
				DL: dl,
			}
			if tt.req == nil {
				tt.req = &furanrpc.BuildStatusRequest{
					BuildId: id.String(),
				}
			}
			got, err := gr.GetBuildStatus(ctx, tt.req)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("GetBuildStatus() error = %v, wantErr %v", err, tt.wantErr)
				}
				if code := status.Code(err); code != tt.wantErrCode {
					t.Errorf("bad error code: %v (wanted %v)", code, tt.wantErrCode)
				}
				return
			}
			if got.BuildId != id.String() {
				t.Errorf("bad ID: %v (wanted %v)", got.BuildId, id)
			}
			if tt.b != nil {
				if got.State != tt.b.Status.State() {
					t.Errorf("bad state: %v (wanted %v)", got.State, tt.b.Status.State())
				}
				if got.Started.Seconds != int64(tt.b.Created.Second()) {
					t.Errorf("bad started seconds: %v (wanted %v)", got.Started.Seconds, tt.b.Created.Second())
				}
			}
		})
	}
}

func TestServer_MonitorBuild(t *testing.T) {
	tests := []struct {
		name string
		// set up the data layer and return the build id
		prepfunc func(dl datalayer.DataLayer) uuid.UUID
		// runs async and produces build messages, then concludes the build
		buildfunc func(dl datalayer.DataLayer, id uuid.UUID)
		// runs async and represents the RPC client
		clientfunc func(msa *MonitorStreamAdapter)
		// runs after the RPC finishes and verifies the data is set correctly
		verifyfunc func(dl datalayer.DataLayer, id uuid.UUID) error
		wantErr    bool
	}{
		{
			name: "success",
			prepfunc: func(dl datalayer.DataLayer) uuid.UUID {
				id, _ := dl.CreateBuild(context.Background(), models.Build{})
				return id
			},
			buildfunc: func(dl datalayer.DataLayer, id uuid.UUID) {
				ctx := context.Background()
				msgs := []string{
					"asdf",
					"asdf2",
					"asdf3",
				}
				for _, m := range msgs {
					dl.AddEvent(ctx, id, m)
					time.Sleep(1 * time.Millisecond)
				}
				dl.SetBuildAsCompleted(ctx, id, models.BuildStatusSuccess)
			},
			clientfunc: func(msa *MonitorStreamAdapter) {
				// we don't know exactly how many msgs will be received by the client
				// it's timing-dependent
				msgs := []*furanrpc.BuildEvent{}
				defer func() {
					t.Logf("clientfunc: %v msgs\n", len(msgs))
				}()
				for {
					msg := &furanrpc.BuildEvent{}
					if err := msa.RecvMsg(msg); err != nil {
						t.Logf("clientfunc: error receiving message: %v", err)
						return
					}
					t.Logf("clientfunc: msg: %+v", msg)
					msgs = append(msgs, msg)
				}
			},
			verifyfunc: func(dl datalayer.DataLayer, id uuid.UUID) error {
				b, _ := dl.GetBuildByID(context.Background(), id)
				if len(b.Events) != 3 {
					return fmt.Errorf("expected 3 events, got %v", len(b.Events))
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dl := &datalayer.FakeDataLayer{}
			id := tt.prepfunc(dl)
			gr := &Server{
				DL: dl,
			}
			ctx, cf := context.WithCancel(context.Background())
			defer cf()
			go tt.buildfunc(dl, id)
			msa := &MonitorStreamAdapter{
				Ctx:        ctx,
				CancelFunc: cf,
			}
			go tt.clientfunc(msa)
			req := &furanrpc.BuildStatusRequest{
				BuildId: id.String(),
			}
			if err := gr.MonitorBuild(req, msa); (err != nil) != tt.wantErr {
				t.Errorf("MonitorBuild() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := tt.verifyfunc(dl, id); err != nil {
				t.Errorf("verify failed: %v", err)
			}
		})
	}
}

func TestServer_CancelBuild(t *testing.T) {
	tests := []struct {
		name        string
		idstr       string
		b           *models.Build
		status      models.BuildStatus
		wantErr     bool
		wantErrCode codes.Code
	}{
		{
			name: "success",
			b: &models.Build{
				GitHubRepo: "acme/foo",
				GitHubRef:  "master",
			},
			status:  models.BuildStatusRunning,
			wantErr: false,
		},
		{
			name: "not cancellable",
			b: &models.Build{
				GitHubRepo: "acme/foo",
				GitHubRef:  "master",
			},
			status:      models.BuildStatusNotStarted,
			wantErr:     true,
			wantErrCode: codes.FailedPrecondition,
		},
		{
			name: "already cancelled",
			b: &models.Build{
				GitHubRepo: "acme/foo",
				GitHubRef:  "master",
			},
			status:      models.BuildStatusCancelled,
			wantErr:     true,
			wantErrCode: codes.FailedPrecondition,
		},
		{
			name:        "not found",
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name:        "bad id",
			idstr:       "invalidid",
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			dl := &datalayer.FakeDataLayer{}
			var id uuid.UUID
			if tt.b == nil {
				id = uuid.Must(uuid.NewV4())
			} else {
				id, _ = dl.CreateBuild(ctx, *tt.b)
				if tt.status != models.BuildStatusUnknown {
					dl.SetBuildStatus(ctx, id, tt.status)
				}
			}
			gr := &Server{
				DL: dl,
			}
			if tt.idstr == "" {
				tt.idstr = id.String()
			}
			resp, err := gr.CancelBuild(ctx, &furanrpc.BuildCancelRequest{
				BuildId: tt.idstr,
			})
			if err != nil {
				if !tt.wantErr {
					t.Errorf("CancelBuild() error = %v, wantErr %v", err, tt.wantErr)
				}
				if code := status.Code(err); code != tt.wantErrCode {
					t.Errorf("bad error code: %v (wanted %v)", code, tt.wantErrCode)
				}
				return
			}
			b, _ := dl.GetBuildByID(ctx, id)
			if b.Status != models.BuildStatusCancelRequested {
				t.Errorf("bad status %v, wanted CancelRequested", b.Status)
			}
			if resp.BuildId != id.String() {
				t.Errorf("bad id %v, wanted %v", resp.BuildId, id)
			}
		})
	}
}

func TestServer_apiKeyAuth(t *testing.T) {
	type fields struct {
		dlsetup func(dl datalayer.DataLayer) uuid.UUID
	}
	type args struct {
		rpcname string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		ctxf   func(id uuid.UUID) context.Context
		want   bool
	}{
		{
			name: "authorized",
			fields: fields{
				dlsetup: func(dl datalayer.DataLayer) uuid.UUID {
					id, _ := dl.CreateAPIKey(context.Background(), models.APIKey{Name: "foo", ReadOnly: false})
					return id
				},
			},
			args: args{
				rpcname: "/furanrpc.FuranExecutor/StartBuild",
			},
			ctxf: func(id uuid.UUID) context.Context {
				md := make(metadata.MD, 1)
				md[APIKeyLabel] = []string{id.String()}
				return metadata.NewIncomingContext(context.Background(), md)
			},
			want: true,
		},
		{
			name: "authorized cancel",
			fields: fields{
				dlsetup: func(dl datalayer.DataLayer) uuid.UUID {
					id, _ := dl.CreateAPIKey(context.Background(), models.APIKey{Name: "foo", ReadOnly: false})
					return id
				},
			},
			args: args{
				rpcname: "/furanrpc.FuranExecutor/CancelBuild",
			},
			ctxf: func(id uuid.UUID) context.Context {
				md := make(metadata.MD, 1)
				md[APIKeyLabel] = []string{id.String()}
				return metadata.NewIncomingContext(context.Background(), md)
			},
			want: true,
		},
		{
			name: "read only authorized",
			fields: fields{
				dlsetup: func(dl datalayer.DataLayer) uuid.UUID {
					id, _ := dl.CreateAPIKey(context.Background(), models.APIKey{Name: "foo", ReadOnly: true})
					return id
				},
			},
			args: args{
				rpcname: "/furanrpc.FuranExecutor/GetBuildStatus",
			},
			ctxf: func(id uuid.UUID) context.Context {
				md := make(metadata.MD, 1)
				md[APIKeyLabel] = []string{id.String()}
				return metadata.NewIncomingContext(context.Background(), md)
			},
			want: true,
		},
		{
			name: "read only unauthorized",
			fields: fields{
				dlsetup: func(dl datalayer.DataLayer) uuid.UUID {
					id, _ := dl.CreateAPIKey(context.Background(), models.APIKey{Name: "foo", ReadOnly: true})
					return id
				},
			},
			args: args{
				rpcname: "/furanrpc.FuranExecutor/StartBuild",
			},
			ctxf: func(id uuid.UUID) context.Context {
				md := make(metadata.MD, 1)
				md[APIKeyLabel] = []string{id.String()}
				return metadata.NewIncomingContext(context.Background(), md)
			},
			want: false,
		},
		{
			name: "unknown key",
			fields: fields{
				dlsetup: func(dl datalayer.DataLayer) uuid.UUID {
					return uuid.Must(uuid.NewV4())
				},
			},
			args: args{
				rpcname: "/furanrpc.FuranExecutor/StartBuild",
			},
			ctxf: func(id uuid.UUID) context.Context {
				md := make(metadata.MD, 1)
				md[APIKeyLabel] = []string{id.String()}
				return metadata.NewIncomingContext(context.Background(), md)
			},
			want: false,
		},
		{
			name: "missing key",
			fields: fields{
				dlsetup: func(dl datalayer.DataLayer) uuid.UUID {
					return uuid.UUID{}
				},
			},
			args: args{
				rpcname: "/furanrpc.FuranExecutor/StartBuild",
			},
			ctxf: func(id uuid.UUID) context.Context {
				return context.Background()
			},
			want: false,
		},
		{
			name: "multiple keys",
			fields: fields{
				dlsetup: func(dl datalayer.DataLayer) uuid.UUID {
					id, _ := dl.CreateAPIKey(context.Background(), models.APIKey{Name: "foo", ReadOnly: false})
					return id
				},
			},
			args: args{
				rpcname: "/furanrpc.FuranExecutor/StartBuild",
			},
			ctxf: func(id uuid.UUID) context.Context {
				md := make(metadata.MD, 2)
				md[APIKeyLabel] = []string{
					id.String(),
					uuid.Must(uuid.NewV4()).String(),
				}
				return metadata.NewIncomingContext(context.Background(), md)
			},
			want: false,
		},
		{
			name: "invalid key",
			fields: fields{
				dlsetup: func(dl datalayer.DataLayer) uuid.UUID {
					return uuid.UUID{}
				},
			},
			args: args{
				rpcname: "/furanrpc.FuranExecutor/StartBuild",
			},
			ctxf: func(id uuid.UUID) context.Context {
				md := make(metadata.MD, 1)
				md[APIKeyLabel] = []string{"some_invalid_key"}
				return metadata.NewIncomingContext(context.Background(), md)
			},
			want: false,
		},
		{
			name: "invalid rpc",
			fields: fields{
				dlsetup: func(dl datalayer.DataLayer) uuid.UUID {
					id, _ := dl.CreateAPIKey(context.Background(), models.APIKey{Name: "foo", ReadOnly: false})
					return id
				},
			},
			args: args{
				rpcname: "/furanrpc.FuranExecutor/SomeInvalidRPC",
			},
			ctxf: func(id uuid.UUID) context.Context {
				md := make(metadata.MD, 1)
				md[APIKeyLabel] = []string{id.String()}
				return metadata.NewIncomingContext(context.Background(), md)
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dl := &datalayer.FakeDataLayer{}
			var id uuid.UUID
			if tt.fields.dlsetup != nil {
				id = tt.fields.dlsetup(dl)
			}
			gr := &Server{
				DL: dl,
				Opts: Options{
					LogFunc: t.Logf,
				},
			}
			s := grpc.NewServer()
			furanrpc.RegisterFuranExecutorServer(s, gr)
			methods, err := methodsFromFuranService(s)
			if err != nil {
				t.Errorf("error getting methods: %v", err)
			}
			gr.methods = methods
			if got := gr.apiKeyAuth(tt.ctxf(id), tt.args.rpcname); got != tt.want {
				t.Errorf("apiKeyAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}
