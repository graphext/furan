package secrets

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/dollarshaveclub/pvc"

	"github.com/dollarshaveclub/furan/v2/pkg/config"
)

type BackendType int

const (
	UnknownBackend BackendType = iota
	VaultBackend
	JSONBackend
	EnvVarBackend
	FileTreeBackend
)

type SecretsClient interface {
	Get(id string) ([]byte, error)
	Fill(s interface{}) error
}

type Fetcher struct {
	Backend                BackendType
	Mapping                string
	JSONFile, FileTreeRoot string
	VaultOptions           config.VaultConfig
	sc                     SecretsClient
	mtx                    sync.Mutex
}

// sanity check against accidentally reading a huge file
const maxJWTSizeBytes = 4096

func (f *Fetcher) init() error {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	var ops []pvc.SecretsClientOption
	switch f.Backend {
	case VaultBackend:
		if !f.VaultOptions.TokenAuth && !f.VaultOptions.K8sAuth {
			return fmt.Errorf("at least one vault auth option is required")
		}
		if f.VaultOptions.TokenAuth && f.VaultOptions.K8sAuth {
			return fmt.Errorf("only one vault auth option is allowed")
		}
		var vauth pvc.VaultAuthentication
		if f.VaultOptions.TokenAuth {
			vauth = pvc.TokenVaultAuth
			ops = append(ops, pvc.WithVaultToken(f.VaultOptions.Token))
		}
		if f.VaultOptions.K8sAuth {
			vauth = pvc.K8sVaultAuth
			f2, err := os.Open(f.VaultOptions.K8sJWTPath)
			if err != nil {
				return fmt.Errorf("error opening k8s jwt: %w", err)
			}
			defer f2.Close()
			fi, err := f2.Stat()
			if err != nil {
				return fmt.Errorf("error stating k8s jwt: %w", err)
			}
			if fi.Size() > maxJWTSizeBytes {
				return fmt.Errorf("k8s jwt exceeds max size (%v): %v", maxJWTSizeBytes, fi.Size())
			}
			jwt, err := ioutil.ReadAll(f2)
			if err != nil {
				return fmt.Errorf("error reading k8s jwt: %w", err)
			}
			ops = append(ops,
				pvc.WithVaultK8sAuth(string(jwt), f.VaultOptions.K8sRole),
				pvc.WithVaultK8sAuthPath(f.VaultOptions.K8sAuthPath),
			)
		}
		ops = append(ops, pvc.WithVaultBackend(vauth, f.VaultOptions.Addr))
	case JSONBackend:
		ops = append(ops, pvc.WithJSONFileBackend(f.JSONFile))
	case EnvVarBackend:
		ops = append(ops, pvc.WithEnvVarBackend())
	case FileTreeBackend:
		ops = append(ops, pvc.WithFileTreeBackend(f.FileTreeRoot))
	default:
		return fmt.Errorf("unknown backend type: %v", f.Backend)
	}
	ops = append(ops, pvc.WithMapping(f.Mapping))
	sc, err := pvc.NewSecretsClient(ops...)
	if err != nil {
		return fmt.Errorf("error creating secrets client: %w", err)
	}
	f.sc = sc
	return nil
}

func (f *Fetcher) GitHub(gc *config.GitHubConfig) error {
	if f.sc == nil {
		if err := f.init(); err != nil {
			return err
		}
	}
	return f.sc.Fill(gc)
}

func (f *Fetcher) Quay(qc *config.QuayConfig) error {
	if f.sc == nil {
		if err := f.init(); err != nil {
			return err
		}
	}
	return f.sc.Fill(qc)
}

func (f *Fetcher) GCR(gc *config.GCRConfig) error {
	if f.sc == nil {
		if err := f.init(); err != nil {
			return err
		}
	}
	return f.sc.Fill(gc)
}

func (f *Fetcher) AWS(ac *config.AWSConfig) error {
	if f.sc == nil {
		if err := f.init(); err != nil {
			return err
		}
	}
	return f.sc.Fill(ac)
}

func (f *Fetcher) Database(dc *config.DBConfig) error {
	if f.sc == nil {
		if err := f.init(); err != nil {
			return err
		}
	}
	if err := f.sc.Fill(dc); err != nil {
		return err
	}
	if i := len(dc.CredentialEncryptionKey); i != 32 {
		return fmt.Errorf("bad size for credential encryption key: %v (wanted 32)", i)
	}
	copy(dc.CredEncKeyArray[:], dc.CredentialEncryptionKey)
	dc.CredentialEncryptionKey = nil
	return nil
}
