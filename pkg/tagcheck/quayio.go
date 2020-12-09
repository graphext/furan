package tagcheck

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// QuayAPIEndpoint is the quay.io API endpoint used to look up tags: https://docs.quay.io/api/swagger/#!/tag
// - Any successful response indicates the tag exists
// - A 404 response means the tag does not exist
// This API call requires a valid OAuth bearer token with repo:read permissions
var QuayAPIEndpoint = "https://quay.io/api/v1/repository/%s/tag/%s/images"

type QuayChecker struct {
	APIToken, endpoint string
	hc                 http.Client
}

// IsQuay returns whether repo is hosted on quay.io
func (qc QuayChecker) IsQuay(repo string) bool {
	return strings.HasPrefix(repo, "quay.io/")
}

func (qc QuayChecker) checkTag(repo, tag string) (bool, error) {
	if qc.endpoint == "" {
		qc.endpoint = QuayAPIEndpoint
	}
	reponame := strings.Replace(repo, "quay.io/", "", 1)
	route := fmt.Sprintf(qc.endpoint, reponame, tag)
	r, err := http.NewRequest("GET", route, nil)
	if err != nil {
		return false, fmt.Errorf("error creating http request: %w", err)
	}
	r.Header.Add("Authorization", "Bearer "+qc.APIToken)
	resp, err := qc.hc.Do(r)
	if err != nil {
		return false, fmt.Errorf("error performing API call: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		b, _ := ioutil.ReadAll(resp.Body)
		return false, fmt.Errorf("status code indicates error: %v: %v", resp.StatusCode, string(b))
	}
}

// AllTagsExist returns whether all tags exist in repo (quay.io/[repo name]), and returns missing tags (if any)
func (qc QuayChecker) AllTagsExist(tags []string, repo string) (bool, []string, error) {
	if !qc.IsQuay(repo) {
		return false, nil, fmt.Errorf("not a quay.io repo")
	}
	missing := []string{}
	for _, t := range tags {
		present, err := qc.checkTag(repo, t)
		if err != nil {
			return false, nil, fmt.Errorf("error checking tag: %v: %w", t, err)
		}
		if !present {
			missing = append(missing, t)
		}
	}
	return len(missing) == 0, missing, nil
}
