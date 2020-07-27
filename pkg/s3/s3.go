package s3

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/mholt/archiver/v3"

	"github.com/dollarshaveclub/furan/pkg/datalayer"
	"github.com/dollarshaveclub/furan/pkg/models"
)

type CacheManager struct {
	AccessKeyID, SecretAccessKey string // AWS credentials scoped to S3 only
	Region, Bucket, Keypfx       string
	DL                           datalayer.DataLayer
	S3UploaderFactoryFunc        func(s *session.Session) s3manageriface.UploaderAPI
	S3DownloaderFactoryFunc      func(s *session.Session) s3manageriface.DownloaderAPI
}

var _ models.CacheFetcher = &CacheManager{}

func (cm *CacheManager) dlclient() (s3manageriface.DownloaderAPI, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      &cm.Region,
		Credentials: credentials.NewStaticCredentials(cm.AccessKeyID, cm.SecretAccessKey, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("error getting aws session: %v", err)
	}
	if cm.S3DownloaderFactoryFunc == nil {
		cm.S3DownloaderFactoryFunc = func(sess *session.Session) s3manageriface.DownloaderAPI {
			return s3manager.NewDownloader(sess)
		}
	}
	return cm.S3DownloaderFactoryFunc(sess), nil
}

func (cm *CacheManager) ulclient() (s3manageriface.UploaderAPI, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      &cm.Region,
		Credentials: credentials.NewStaticCredentials(cm.AccessKeyID, cm.SecretAccessKey, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("error getting aws session: %v", err)
	}
	if cm.S3UploaderFactoryFunc == nil {
		cm.S3UploaderFactoryFunc = func(sess *session.Session) s3manageriface.UploaderAPI {
			return s3manager.NewUploader(sess)
		}
	}
	return cm.S3UploaderFactoryFunc(sess), nil
}

func (cm *CacheManager) keyForBuild(b models.Build) string {
	// Example key: keyprefix/acmecorp/widgets_buildcache.tar.gz
	return cm.Keypfx + b.GitHubRepo + "_buildcache.tar.gz"
}

// Fetch fetches the cache for a build from S3 and returns the temporary filesystem path where it can be found.
// Caller is responsible for cleaning the path up when finished with the data.
func (cm *CacheManager) Fetch(ctx context.Context, b models.Build) (string, error) {
	dlc, err := cm.dlclient()
	if err != nil {
		return "", fmt.Errorf("error getting S3 downloader: %w", err)
	}
	f, err := ioutil.TempFile("", "s3-cache-fetch-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("error creating temp file: %w", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()
	key := cm.keyForBuild(b)
	start := time.Now().UTC()
	n, err := dlc.DownloadWithContext(ctx, f, &s3.GetObjectInput{
		Bucket: &cm.Bucket,
		Key:    &key,
	})
	if err != nil {
		return "", fmt.Errorf("error downloading cache from S3: %w", err)
	}
	cm.DL.AddEvent(ctx, b.ID, fmt.Sprintf("successfully downloaded cache from S3 (size %v bytes; elapsed: %v)", n, time.Since(start)))
	start = time.Now().UTC()
	dir, err := ioutil.TempDir("", "build-cache-*")
	if err != nil {
		return "", fmt.Errorf("error getting temp dir: %w", err)
	}
	if err := archiver.Unarchive(f.Name(), dir); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("error unarchiving build cache: %w", err)
	}
	cm.DL.AddEvent(ctx, b.ID, fmt.Sprintf("successfully unarchived build cache (elapsed: %v)", time.Since(start)))
	return dir, nil
}

// Save persists the cache for a build located at path in S3.
// Caller is responsible for cleaning up the path after Save returns.
func (cm *CacheManager) Save(ctx context.Context, b models.Build, path string) error {
	ulc, err := cm.ulclient()
	if err != nil {
		return fmt.Errorf("error getting s3 uploader: %w", err)
	}
	f, err := ioutil.TempFile("", "s3-cache-save-*.tar.gz")
	if err != nil {
		return fmt.Errorf("error getting temp file: %w", err)
	}
	f.Close()
	defer os.Remove(f.Name())
	tgz := archiver.NewTarGz()
	tgz.OverwriteExisting = true
	start := time.Now().UTC()
	if err := tgz.Archive([]string{path}, f.Name()); err != nil {
		return fmt.Errorf("error archiving build cache: %w", err)
	}
	cm.DL.AddEvent(ctx, b.ID, fmt.Sprintf("archived build cache (elapsed: %v)", time.Since(start)))
	f, err = os.Open(f.Name())
	if err != nil {
		return fmt.Errorf("error opening archive: %w", err)
	}
	var sz int64
	fi, err := f.Stat()
	if err == nil {
		sz = fi.Size()
	}
	defer f.Close()
	key := cm.keyForBuild(b)
	start = time.Now().UTC()
	if _, err := ulc.UploadWithContext(ctx, &s3manager.UploadInput{
		Body:   f,
		Bucket: &cm.Bucket,
		Key:    &key,
	}); err != nil {
		return fmt.Errorf("error uploading build cache: %w", err)
	}
	cm.DL.AddEvent(ctx, b.ID, fmt.Sprintf("successfully uploaded cache from S3 (size %v bytes; elapsed: %v)", sz, time.Since(start)))
	return nil
}
