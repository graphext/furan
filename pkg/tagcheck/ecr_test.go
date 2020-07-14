package tagcheck

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	awsecr "github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecr/ecriface"
)

func TestRegistryManager_IsECR(t *testing.T) {
	type fields struct {
		AccessKeyID          string
		SecretAccessKey      string
		ecrClientFactoryFunc func(s *session.Session) ecriface.ECRAPI
	}
	type args struct {
		repo string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "ecr repo",
			args: args{repo: "123456789.dkr.ecr.us-west-2.amazonaws.com/widgets"},
			want: true,
		},
		{
			name: "ecr repo with namespace",
			args: args{repo: "123456789.dkr.ecr.us-west-2.amazonaws.com/acme/widgets"},
			want: true,
		},
		{
			name: "ecr repo with tag",
			args: args{repo: "123456789.dkr.ecr.us-west-2.amazonaws.com/widgets:master"},
			want: true,
		},
		{
			name: "non-ecr repo (quay)",
			args: args{repo: "quay.io/acme/foobar"},
			want: false,
		},
		{
			name: "non-ecr repo (docker hub)",
			args: args{repo: "acme/foobar"},
			want: false,
		},
		{
			name: "non-ecr repo with tag (quay)",
			args: args{repo: "quay.io/acme/foobar:master"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ECRChecker{
				AccessKeyID:          tt.fields.AccessKeyID,
				SecretAccessKey:      tt.fields.SecretAccessKey,
				ECRClientFactoryFunc: tt.fields.ecrClientFactoryFunc,
			}
			if got := r.IsECR(tt.args.repo); got != tt.want {
				t.Errorf("IsECR() = %v, want %v", got, tt.want)
			}
		})
	}
}

func sliceContentsEqual(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}
	m := make(map[string]struct{}, len(s1))
	for i := range s1 {
		m[s1[i]] = struct{}{}
	}
	for _, s := range s2 {
		if _, ok := m[s]; !ok {
			return false
		}
	}
	return true
}

type FakeECRClient struct {
	ecriface.ECRAPI
	DescribeImagesPagesFunc func(input *awsecr.DescribeImagesInput, fn func(*awsecr.DescribeImagesOutput, bool) bool) error
}

func (fec FakeECRClient) DescribeImagesPages(input *awsecr.DescribeImagesInput, fn func(*awsecr.DescribeImagesOutput, bool) bool) error {
	if fec.DescribeImagesPagesFunc != nil {
		return fec.DescribeImagesPagesFunc(input, fn)
	}
	return nil
}

func TestRegistryManager_AllTagsExist(t *testing.T) {
	type fields struct {
		ecrClientFactoryFunc func(s *session.Session) ecriface.ECRAPI
	}
	type args struct {
		tags []string
		repo string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		want1   []string
		wantErr bool
	}{
		{
			name: "all exist",
			fields: fields{
				ecrClientFactoryFunc: func(_ *session.Session) ecriface.ECRAPI {
					return &FakeECRClient{
						DescribeImagesPagesFunc: func(input *awsecr.DescribeImagesInput, fn func(*awsecr.DescribeImagesOutput, bool) bool) error {
							if len(input.ImageIds) != 1 {
								return fmt.Errorf("expected 1 image id: %v", len(input.ImageIds))
							}
							fn(&awsecr.DescribeImagesOutput{
								ImageDetails: []*awsecr.ImageDetail{
									&awsecr.ImageDetail{
										RepositoryName: input.RepositoryName,
										ImageTags: []*string{
											input.ImageIds[0].ImageTag,
										},
									},
								},
							}, true)
							return nil
						},
					}
				},
			},
			args: args{
				tags: []string{"master", "release", "asdf"},
				repo: "123456789.dkr.ecr.us-west-2.amazonaws.com/widgets",
			},
			want:    true,
			want1:   []string{},
			wantErr: false,
		},
		{
			name: "all exist, repo with namespace",
			fields: fields{
				ecrClientFactoryFunc: func(_ *session.Session) ecriface.ECRAPI {
					return &FakeECRClient{
						DescribeImagesPagesFunc: func(input *awsecr.DescribeImagesInput, fn func(*awsecr.DescribeImagesOutput, bool) bool) error {
							if len(input.ImageIds) != 1 {
								return fmt.Errorf("expected 1 image id: %v", len(input.ImageIds))
							}
							fn(&awsecr.DescribeImagesOutput{
								ImageDetails: []*awsecr.ImageDetail{
									&awsecr.ImageDetail{
										RepositoryName: input.RepositoryName,
										ImageTags: []*string{
											input.ImageIds[0].ImageTag,
										},
									},
								},
							}, true)
							return nil
						},
					}
				},
			},
			args: args{
				tags: []string{"master", "release", "asdf"},
				repo: "123456789.dkr.ecr.us-west-2.amazonaws.com/acme/widgets",
			},
			want:    true,
			want1:   []string{},
			wantErr: false,
		},
		{
			name: "missing tag",
			fields: fields{
				ecrClientFactoryFunc: func(_ *session.Session) ecriface.ECRAPI {
					return &FakeECRClient{
						DescribeImagesPagesFunc: func(input *awsecr.DescribeImagesInput, fn func(*awsecr.DescribeImagesOutput, bool) bool) error {
							if len(input.ImageIds) != 1 {
								return fmt.Errorf("expected 1 image id: %v", len(input.ImageIds))
							}
							if *input.ImageIds[0].ImageTag == "release" {
								return awserr.New(awsecr.ErrCodeImageNotFoundException, "some message", fmt.Errorf("some err"))
							}
							fn(&awsecr.DescribeImagesOutput{
								ImageDetails: []*awsecr.ImageDetail{
									&awsecr.ImageDetail{
										RepositoryName: input.RepositoryName,
										ImageTags: []*string{
											input.ImageIds[0].ImageTag,
										},
									},
								},
							}, true)
							return nil
						},
					}
				},
			},
			args: args{
				tags: []string{"master", "release", "asdf"},
				repo: "123456789.dkr.ecr.us-west-2.amazonaws.com/widgets",
			},
			want:    false,
			want1:   []string{"release"},
			wantErr: false,
		},
		{
			name: "non-ecr repo",
			args: args{
				tags: []string{"master", "release", "asdf"},
				repo: "quay.io/acme/widgets",
			},
			want:    false,
			want1:   []string{},
			wantErr: true,
		},
		{
			name: "malformed repo",
			args: args{
				tags: []string{"master", "release", "asdf"},
				repo: "somerandomthing",
			},
			want:    false,
			want1:   []string{},
			wantErr: true,
		},
		{
			name: "repo with tag",
			args: args{
				tags: []string{"master", "release", "asdf"},
				repo: "123456789.dkr.ecr.us-west-2.amazonaws.com/widgets:master",
			},
			want:    false,
			want1:   []string{},
			wantErr: true,
		},
		{
			name: "aws error",
			fields: fields{
				ecrClientFactoryFunc: func(_ *session.Session) ecriface.ECRAPI {
					return &FakeECRClient{
						DescribeImagesPagesFunc: func(input *awsecr.DescribeImagesInput, fn func(*awsecr.DescribeImagesOutput, bool) bool) error {
							if len(input.ImageIds) != 1 {
								return fmt.Errorf("expected 1 image id: %v", len(input.ImageIds))
							}
							if *input.ImageIds[0].ImageTag == "release" {
								return awserr.New(awsecr.ErrCodeServerException, "some message", fmt.Errorf("some err"))
							}
							fn(&awsecr.DescribeImagesOutput{
								ImageDetails: []*awsecr.ImageDetail{
									&awsecr.ImageDetail{
										RepositoryName: input.RepositoryName,
										ImageTags: []*string{
											input.ImageIds[0].ImageTag,
										},
									},
								},
							}, true)
							return nil
						},
					}
				},
			},
			args: args{
				tags: []string{"master", "release", "asdf"},
				repo: "123456789.dkr.ecr.us-west-2.amazonaws.com/widgets",
			},
			want:    false,
			want1:   []string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ECRChecker{
				ECRClientFactoryFunc: tt.fields.ecrClientFactoryFunc,
			}
			got, got1, err := r.AllTagsExist(tt.args.tags, tt.args.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("AllTagsExist() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AllTagsExist() got = %v, want %v", got, tt.want)
			}
			if !sliceContentsEqual(got1, tt.want1) {
				t.Errorf("AllTagsExist() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
