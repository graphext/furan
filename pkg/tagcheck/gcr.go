package tagcheck

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// - Any successful response indicates the tag exists
// - A 404 response means the tag does not exist
// This API call requires a valid OAuth bearer token with repo:read permissions
var GCRAPIEndpoint = "https://eu.gcr.io/v2/%s/tags/list"

type GCRChecker struct {
	ServiceAccount, endpoint string
	hc                       http.Client
}

type GCRTagsList struct {
	Tags []string `json:"tags"`
}

// IsGCR returns whether repo is hosted on eu.gcr.io
func (gc GCRChecker) IsGCR(repo string) bool {
	return strings.HasPrefix(repo, "eu.gcr.io/")
}

func (gc GCRChecker) checkTag(repo, tag string) (bool, error) {
	if gc.endpoint == "" {
		gc.endpoint = GCRAPIEndpoint
	}

	reponame := strings.Replace(repo, "eu.gcr.io/", "", 1)
	route := fmt.Sprintf(gc.endpoint, reponame)

	r, err := http.NewRequest("GET", route, nil)
	if err != nil {
		return false, fmt.Errorf("error creating http request: %w", err)
	}

	r.SetBasicAuth("_json_key", gc.ServiceAccount)

	resp, err := gc.hc.Do(r)
	if err != nil {
		return false, fmt.Errorf("error performing API call: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		b, _ := ioutil.ReadAll(resp.Body)
		var gcrTags GCRTagsList
		err := json.Unmarshal(b, &gcrTags)
		if err != nil {
			return false, fmt.Errorf("error parsing returned tags: %w", err)
		}
		var tag_found bool = false
		for _, v := range gcrTags.Tags {
			if v == tag {
				tag_found = true
			}
		}
		return tag_found, nil
	case http.StatusNotFound:
		return false, nil
	default:
		b, _ := ioutil.ReadAll(resp.Body)
		return false, fmt.Errorf("status code indicates error: %v: %v", resp.StatusCode, string(b))
	}
}

// AllTagsExist returns whether all tags exist in repo (eu.gcr.io/[repo name]), and returns missing tags (if any)
func (gc GCRChecker) AllTagsExist(tags []string, repo string) (bool, []string, error) {
	if !gc.IsGCR(repo) {
		return false, nil, fmt.Errorf("not a eu.gcr.io repo")
	}
	missing := []string{}
	for _, t := range tags {
		present, err := gc.checkTag(repo, t)
		if err != nil {
			return false, nil, fmt.Errorf("error checking tag: %v: %w", t, err)
		}
		if !present {
			missing = append(missing, t)
		}
	}
	return len(missing) == 0, missing, nil
}
