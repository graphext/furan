package tagcheck

import "fmt"

type Checker struct {
	Quay *QuayChecker
	ECR  *ECRChecker
}

func (c *Checker) AllTagsExist(tags []string, repo string) (bool, []string, error) {
	switch {
	case c.ECR.IsECR(repo):
		return c.ECR.AllTagsExist(tags, repo)
	case c.Quay.IsQuay(repo):
		return c.Quay.AllTagsExist(tags, repo)
	}
	return false, nil, fmt.Errorf("unsupported or bad repo: %v", repo)
}
