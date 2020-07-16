package models

import (
	"crypto/rand"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"golang.org/x/crypto/nacl/secretbox"

	"github.com/dollarshaveclub/furan/pkg/generated/furanrpc"
)

//go:generate stringer -type=BuildStatus

type BuildStatus int

const (
	BuildStatusUnknown BuildStatus = iota
	BuildStatusNotStarted
	BuildStatusSkipped
	BuildStatusRunning
	BuildStatusFailure
	BuildStatusSuccess
	BuildStatusCancelled
)

type Build struct {
	ID                          uuid.UUID
	Created, Updated, Completed time.Time
	GitHubRepo, GitHubRef       string
	EncryptedGitHubCredential   []byte
	ImageRepos                  []string
	Tags                        []string
	CommitSHATag                bool
	DisableBuildCache           bool
	Request                     furanrpc.BuildRequest
	Status                      BuildStatus
	Events                      []string
}

func (b Build) CanAddEvent() bool {
	return b.Running()
}

func (b Build) Running() bool {
	return b.Status == BuildStatusRunning
}

// EncryptAndSetGitHubCredential takes a GitHub credential, encrypts it and sets EncryptedGitHubCredential accordingly
func (b *Build) EncryptAndSetGitHubCredential(cred []byte, key [32]byte) error {
	var nonce [24]byte
	if n, err := rand.Read(nonce[:]); err != nil || n != len(nonce) {
		return errors.Wrapf(err, "error reading random bytes for nonce (read: %v)", n)
	}
	b.EncryptedGitHubCredential = secretbox.Seal(nonce[:], cred, &nonce, &key)
	return nil
}

// GetGitHubCredential returns the decrypted user token using key or error
func (b Build) GetGitHubCredential(key [32]byte) (string, error) {
	var nonce [24]byte
	copy(nonce[:], b.EncryptedGitHubCredential[:24])
	tkn, ok := secretbox.Open(nil, b.EncryptedGitHubCredential[24:], &nonce, &key)
	if !ok {
		return "", errors.New("decryption error (incorrect key?)")
	}
	return string(tkn), nil
}

//go:generate stringer -type=BuildCacheType

type BuildCacheType int

const (
	UnknownCacheType BuildCacheType = iota
	DisabledCacheType
	InlineCacheType
	S3CacheType
)

type CacheOpts struct {
	Type    BuildCacheType
	MaxMode bool
}

// BuildOpts models all options required to perform a build
type BuildOpts struct {
	BuildID                uuid.UUID
	ContextPath, CommitSHA string
	RelativeDockerfilePath string
	BuildArgs              map[string]string
	Cache                  CacheOpts
}

// Job describes methods on a single abstract build job
type Job interface {
	// Close cleans up any resources associated with this Job
	Close()
	// Error returns a channel that will contain any errors associated with this Job
	Error() chan error
	// Running returns a channel that signals that the build the Job is executing has been updated to status Running
	// This indicates that the Furan sidecar has started and is executing successfully and will take responsibility for
	// tracking the build status from this point forward
	Running() chan struct{}
	// Logs returns all pod logs associated with the Job
	Logs() (map[string]map[string][]byte, error)
}

// CacheFetcher describes an object that fetches and saves build cache
type CacheFetcher interface {
	// Fetch fetches the build cache for a build and returns a local filesystem
	// path where it was written. Caller is responsible for cleaning up the path when finished.
	Fetch(b Build) (string, error)
	// Save persists the build cache for a build located at path.
	// Caller is responsible for cleaning up the path afterward.
	Save(b Build, path string) error
}
