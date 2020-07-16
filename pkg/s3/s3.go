package s3

import "github.com/dollarshaveclub/furan/pkg/models"

type CacheManager struct {
}

var _ models.CacheFetcher = &CacheManager{}

func (cm *CacheManager) Fetch(b models.Build) (string, error) {
	return "", nil
}

func (cm *CacheManager) Save(b models.Build, path string) error {
	return nil
}
