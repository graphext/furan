package s3

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"

	"github.com/dollarshaveclub/furan/v2/pkg/datalayer"
	"github.com/dollarshaveclub/furan/v2/pkg/models"
)

type fakeDownloader struct {
	s3manageriface.DownloaderAPI
	DownloadWithContextFunc func(aws.Context, io.WriterAt, *s3.GetObjectInput, ...func(*s3manager.Downloader)) (int64, error)
}

func (fd *fakeDownloader) DownloadWithContext(ctx aws.Context, f io.WriterAt, in *s3.GetObjectInput, opts ...func(*s3manager.Downloader)) (int64, error) {
	if fd.DownloadWithContextFunc != nil {
		return fd.DownloadWithContextFunc(ctx, f, in, opts...)
	}
	return 0, nil
}

func TestCacheManager_Fetch(t *testing.T) {
	type fields struct {
		Region string
		Bucket string
		Keypfx string
	}
	type args struct {
		b models.Build
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		datapath    string
		downloaderr error
		verifyf     func(path string) error
		wantErr     bool
	}{
		{
			name: "success",
			fields: fields{
				Region: "us-west-2",
				Bucket: "foobucket",
				Keypfx: "somerandom/path/",
			},
			args: args{
				b: models.Build{
					GitHubRepo: "acme/microservice",
				},
			},
			datapath: "testdata/test.tar.gz",
			verifyf: func(path string) error {
				f, err := os.Open(path)
				if err != nil {
					return fmt.Errorf("error opening: %v", err)
				}
				defer f.Close()
				files, err := f.Readdir(-1)
				if err != nil {
					return fmt.Errorf("error reading dir: %v", err)
				}
				if i := len(files); i != 1 {
					return fmt.Errorf("expected one subdirectory, got %v", i)
				}
				if !files[0].IsDir() {
					return fmt.Errorf("expected a subdirectory but isn't a dir: %v", files[0].Name())
				}
				if n := files[0].Name(); n != "foo" {
					return fmt.Errorf("expected a subdirectory named foo, got %v", files[0].Name())
				}
				return nil
			},
		},
		{
			name: "error",
			fields: fields{
				Region: "us-west-2",
				Bucket: "foobucket",
				Keypfx: "somerandom/path/",
			},
			args: args{
				b: models.Build{
					GitHubRepo: "acme/microservice",
				},
			},
			downloaderr: fmt.Errorf("error downloading"),
			wantErr:     true,
			verifyf: func(path string) error {
				if path != "" {
					return fmt.Errorf("unexpected path: %v", path)
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &CacheManager{
				DL:     &datalayer.FakeDataLayer{},
				Region: tt.fields.Region,
				Bucket: tt.fields.Bucket,
				Keypfx: tt.fields.Keypfx,
			}
			s3df := func(s *session.Session) s3manageriface.DownloaderAPI {
				if s.Config.Region == nil {
					t.Errorf("aws session region is nil")
					return nil
				}
				if *s.Config.Region != tt.fields.Region {
					t.Errorf("bad aws region: %v (wanted %v)", *s.Config.Region, tt.fields.Region)
					return nil
				}
				return &fakeDownloader{
					DownloadWithContextFunc: func(ctx aws.Context, f io.WriterAt, in *s3.GetObjectInput, opts ...func(*s3manager.Downloader)) (int64, error) {
						if in == nil || in.Bucket == nil || in.Key == nil || f == nil {
							return 0, fmt.Errorf("one or more inputs are nil")
						}
						if *in.Bucket != tt.fields.Bucket {
							return 0, fmt.Errorf("bad bucket: %v (wanted %v)", in.Bucket, tt.fields.Bucket)
						}
						if k := cm.keyForBuild(tt.args.b); *in.Key != k {
							return 0, fmt.Errorf("bad key: %v (wanted %v)", *in.Key, k)
						}
						if tt.downloaderr != nil {
							return 0, tt.downloaderr
						}
						d, err := ioutil.ReadFile(tt.datapath)
						if err != nil {
							return 0, fmt.Errorf("error reading datapath: %v", err)
						}
						if _, err := f.WriteAt(d, 0); err != nil {
							return 0, fmt.Errorf("error writing data: %v", err)
						}
						return int64(len(d)), nil
					},
				}
			}
			cm.S3DownloaderFactoryFunc = s3df
			got, err := cm.Fetch(context.Background(), tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.verifyf != nil {
				if err := tt.verifyf(got); err != nil {
					t.Errorf("error: %v", err)
				}
			}
		})
	}
}

type fakeUploader struct {
	s3manageriface.UploaderAPI
	UploadWithContextFunc func(aws.Context, *s3manager.UploadInput, ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)
}

func (fd *fakeUploader) UploadWithContext(ctx aws.Context, in *s3manager.UploadInput, opts ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
	if fd.UploadWithContextFunc != nil {
		return fd.UploadWithContextFunc(ctx, in, opts...)
	}
	return &s3manager.UploadOutput{}, nil
}

func TestCacheManager_Save(t *testing.T) {
	type fields struct {
		Region string
		Bucket string
		Keypfx string
	}
	type args struct {
		b models.Build
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		pathfunc  func() string
		uploaderr error
		wantErr   bool
	}{
		{
			name: "success",
			fields: fields{
				Region: "us-west-2",
				Bucket: "foo",
				Keypfx: "some/prefix/",
			},
			args: args{
				b: models.Build{},
			},
			pathfunc: func() string {
				tdir, _ := ioutil.TempDir("", "")
				f, _ := os.Create(filepath.Join(tdir, "foo.txt"))
				f.Write([]byte("asdf1234\n"))
				f.Close()
				return tdir
			},
		},
		{
			name: "no such file",
			fields: fields{
				Region: "us-west-2",
				Bucket: "foo",
				Keypfx: "some/prefix/",
			},
			args: args{
				b: models.Build{},
			},
			pathfunc: func() string {
				return "/path/to/does/not/exist"
			},
			wantErr: true,
		},
		{
			name: "upload error",
			fields: fields{
				Region: "us-west-2",
				Bucket: "foo",
				Keypfx: "some/prefix/",
			},
			args: args{
				b: models.Build{},
			},
			pathfunc: func() string {
				tdir, _ := ioutil.TempDir("", "")
				f, _ := os.Create(filepath.Join(tdir, "foo.txt"))
				f.Write([]byte("asdf1234\n"))
				f.Close()
				return tdir
			},
			uploaderr: fmt.Errorf("upload error happened"),
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &CacheManager{
				DL:     &datalayer.FakeDataLayer{},
				Region: tt.fields.Region,
				Bucket: tt.fields.Bucket,
				Keypfx: tt.fields.Keypfx,
			}
			s3uf := func(s *session.Session) s3manageriface.UploaderAPI {
				if s.Config.Region == nil {
					t.Errorf("aws session region is nil")
					return nil
				}
				if *s.Config.Region != tt.fields.Region {
					t.Errorf("bad aws region: %v (wanted %v)", *s.Config.Region, tt.fields.Region)
					return nil
				}
				return &fakeUploader{
					UploadWithContextFunc: func(ctx aws.Context, in *s3manager.UploadInput, opts ...func(uploader *s3manager.Uploader)) (*s3manager.UploadOutput, error) {
						if in == nil || in.Bucket == nil || in.Key == nil {
							return nil, fmt.Errorf("one or more inputs are nil")
						}
						if *in.Bucket != tt.fields.Bucket {
							return nil, fmt.Errorf("bad bucket: %v (wanted %v)", in.Bucket, tt.fields.Bucket)
						}
						if k := cm.keyForBuild(tt.args.b); *in.Key != k {
							return nil, fmt.Errorf("bad key: %v (wanted %v)", *in.Key, k)
						}
						if tt.uploaderr != nil {
							return nil, tt.uploaderr
						}
						_, err := ioutil.ReadAll(in.Body)
						if err != nil {
							return nil, fmt.Errorf("error reading input: %v", err)
						}
						return &s3manager.UploadOutput{}, nil
					},
				}
			}
			cm.S3UploaderFactoryFunc = s3uf
			p := tt.pathfunc()
			defer os.RemoveAll(p)
			if err := cm.Save(context.Background(), tt.args.b, p); (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
