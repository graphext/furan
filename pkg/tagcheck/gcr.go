package tagcheck

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	googauth "golang.org/x/oauth2/google"
)

// - Any successful response indicates the tag exists
// - A 404 response means the tag does not exist
// This API call requires a valid OAuth bearer token with repo:read permissions
var GCRAPIEndpoint = "https://eu.gcr.io/v2/%s/%s/tags/list"

type GCRChecker struct {
	ServiceAccount, endpoint string
	hc                       http.Client
}

// IsGCR returns whether repo is hosted on eu.gcr.io
func (gc GCRChecker) IsGCR(repo string) bool {
	return strings.HasPrefix(repo, "eu.gcr.io/")
}

type GCRServiceAccount struct {
	ProjectID string `json:"project_id"`
}

var serviceAccount GCRServiceAccount

func (gc GCRChecker) checkTag(repo, tag string) (bool, error) {
	if gc.endpoint == "" {
		gc.endpoint = GCRAPIEndpoint
	}

	err := json.Unmarshal([]byte(gc.ServiceAccount), &serviceAccount)

	if err != nil {
		return false, fmt.Errorf("error parsing gcr credentials: %w", err)
	}

	reponame := strings.Replace(repo, "eu.gcr.io/", "", 1)
	route := fmt.Sprintf(gc.endpoint, serviceAccount.ProjectID, reponame)

	r, err := http.NewRequest("GET", route, nil)
	if err != nil {
		return false, fmt.Errorf("error creating http request: %w", err)
	}

	ts, err := googauth.JWTAccessTokenSourceFromJSON([]byte(gc.ServiceAccount), "")
	if err != nil {
		return false, fmt.Errorf("error getting jwt access token from google: %w", err)
	}

	bearer_token, err := ts.Token()
	if err != nil {
		return false, fmt.Errorf("error getting jwt access token from google: %w", err)
	}

	r.Header.Add("Authorization", "Bearer "+bearer_token.AccessToken)
	resp, err := gc.hc.Do(r)
	if err != nil {
		return false, fmt.Errorf("error performing API call: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		b, _ := ioutil.ReadAll(resp.Body)
		var tags []string
		err := json.Unmarshal(b, &tags)
		if err != nil {
			return false, fmt.Errorf("error parsing returned tags: %w", err)
		}
		var tag_found bool = false
		for _, v := range tags {
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
