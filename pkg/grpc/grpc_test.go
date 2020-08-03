package grpc

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

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
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{Opts: Options{
				TraceSvcName:            "",
				CredentialDecryptionKey: [32]byte{},
				Cache:                   models.CacheOpts{},
				LogFunc:                 nil,
			}},
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			dl := &datalayer.FakeDataLayer{}
			bm := &builder.FakeBuildManager{StartFunc: func(ctx context.Context, opts models.BuildOpts) error {
				return nil
			}}
			gr := &Server{
				DL: dl,
				BM: bm,
				CFFactory: func(token string) models.CodeFetcher {
					return &github.FakeFetcher{}
				},
				Opts: tt.fields.Opts,
			}
			got, err := gr.StartBuild(ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("StartBuild() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil {
				_, err = dl.GetBuildByID(ctx, uuid.Must(uuid.FromString(got.BuildId)))
				if err != nil {
					t.Errorf("error validating build id: %v", err)
				}
			}

		})
	}
}
