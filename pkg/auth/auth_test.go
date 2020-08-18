package auth

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	ecrapi "github.com/awslabs/amazon-ecr-credential-helper/ecr-login/api"
	bkauth "github.com/moby/buildkit/session/auth"
)

type FakeECRAuthClient struct {
	ecrapi.Client
	GetCredsFunc func(host string) (*ecrapi.Auth, error)
}

func (eac FakeECRAuthClient) GetCredentials(host string) (*ecrapi.Auth, error) {
	if eac.GetCredsFunc != nil {
		return eac.GetCredsFunc(host)
	}
	return &ecrapi.Auth{}, nil
}

func TestRegistryManager_GetECRAuth(t *testing.T) {
	type fields struct {
		ECRAuthClientFactoryFunc func(s *awssession.Session, cfg *aws.Config) ecrapi.Client
	}
	type args struct {
		host string
	}
	authfunc := func(s *awssession.Session, cfg *aws.Config) ecrapi.Client {
		return &FakeECRAuthClient{
			GetCredsFunc: func(host string) (*ecrapi.Auth, error) {
				return &ecrapi.Auth{
					Username: "foo",
					Password: "bar",
				}, nil
			},
		}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			name: "ecr",
			fields: fields{
				ECRAuthClientFactoryFunc: authfunc,
			},
			args: args{
				host: "123456789.dkr.ecr.us-west-2.amazonaws.com",
			},
			want:  "foo",
			want1: "bar",
		},
		{
			name: "quay",
			fields: fields{
				ECRAuthClientFactoryFunc: authfunc,
			},
			args: args{
				host: "quay.io",
			},
			wantErr: true,
		},
		{
			name: "something else",
			fields: fields{
				ECRAuthClientFactoryFunc: authfunc,
			},
			args: args{
				host: "example.com",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				ECRAuthClientFactoryFunc: tt.fields.ECRAuthClientFactoryFunc,
			}
			got, got1, err := p.GetECRAuth(tt.args.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetECRAuth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetECRAuth() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetECRAuth() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestProvider_Credentials(t *testing.T) {
	tests := []struct {
		name                     string
		quaytoken                string
		ECRAuthClientFactoryFunc func(s *awssession.Session, cfg *aws.Config) ecrapi.Client
		req                      *bkauth.CredentialsRequest
		want                     *bkauth.CredentialsResponse
		wantErr                  bool
	}{
		{
			name: "ecr",
			ECRAuthClientFactoryFunc: func(s *awssession.Session, cfg *aws.Config) ecrapi.Client {
				return &FakeECRAuthClient{
					GetCredsFunc: func(host string) (*ecrapi.Auth, error) {
						return &ecrapi.Auth{
							Username: "foo",
							Password: "bar",
						}, nil
					},
				}
			},
			req: &bkauth.CredentialsRequest{Host: "123456789.dkr.ecr.us-west-2.amazonaws.com"},
			want: &bkauth.CredentialsResponse{
				Username: "foo",
				Secret:   "bar",
			},
		},
		{
			name:      "quay",
			quaytoken: "asdf1234",
			req:       &bkauth.CredentialsRequest{Host: "quay.io"},
			want: &bkauth.CredentialsResponse{
				Username: "",
				Secret:   "asdf1234",
			},
		},
		{
			name:    "unknown",
			req:     &bkauth.CredentialsRequest{Host: "example.com"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			p := &Provider{
				QuayIOToken:              tt.quaytoken,
				ECRAuthClientFactoryFunc: tt.ECRAuthClientFactoryFunc,
			}
			got, err := p.Credentials(ctx, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Credentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got == nil {
				t.Errorf("got is nil")
				return
			}
			if got.Username != tt.want.Username {
				t.Errorf("Secret: got = %v, want %v", got, tt.want)
			}
			if got.Secret != tt.want.Secret {
				t.Errorf("Secret: got = %v, want %v", got, tt.want)
			}
		})
	}
}
