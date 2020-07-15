package github

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/google/go-github/github"
)

// modified copypasta from https://github.com/google/go-github/blob/master/github/github_test.go
func testGHClient() (*github.Client, *http.ServeMux, string, func()) {
	// mux is the HTTP request multiplexer used with the test server.
	mux := http.NewServeMux()

	apiHandler := http.NewServeMux()
	apiHandler.Handle("/", mux)

	// server is a test HTTP server used to provide mock API responses.
	server := httptest.NewServer(apiHandler)

	// client is the GitHub client being tested and is
	// configured to use test server.
	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL + "/")
	client.BaseURL = url
	client.UploadURL = url
	return client, mux, server.URL, server.Close
}

func TestGitHubFetcher_GetCommitSHA(t *testing.T) {
	type args struct {
		repo string
		ref  string
	}
	tests := []struct {
		name     string
		muxfunc  func(mux *http.ServeMux)
		args     args
		wantCsha string
		wantErr  bool
	}{
		{
			name: "success",
			muxfunc: func(mux *http.ServeMux) {
				mux.HandleFunc("/repos/acme/foo/commits/master", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("01234abcde"))
				})
			},
			args: args{
				repo: "acme/foo",
				ref:  "master",
			},
			wantCsha: "01234abcde",
		},
		{
			name: "not found",
			muxfunc: func(mux *http.ServeMux) {
				mux.HandleFunc("/repos/acme/foo/commits/master", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				})
			},
			args: args{
				repo: "acme/foo",
				ref:  "master",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mux, _, teardown := testGHClient()
			defer teardown()
			if tt.muxfunc != nil {
				tt.muxfunc(mux)
			}
			gf := &GitHubFetcher{
				c: client,
			}
			gotCsha, err := gf.GetCommitSHA(context.Background(), tt.args.repo, tt.args.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCommitSHA() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotCsha != tt.wantCsha {
				t.Errorf("GetCommitSHA() gotCsha = %v, want %v", gotCsha, tt.wantCsha)
			}
		})
	}
}

func TestGitHubFetcher_Fetch(t *testing.T) {
	testtarh := func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open("testdata/test.tar.gz")
		if err != nil {
			t.Logf("error reading test tar: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer f.Close()
		w.Header().Add("Content-Type", "application/octet-stream")
		io.Copy(w, f)
	}
	arclinkh := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Location", "http://"+r.Host+"/archive")
		w.WriteHeader(http.StatusFound)
	}
	type args struct {
		repo string
		ref  string
	}
	tests := []struct {
		name      string
		muxfunc   func(mux *http.ServeMux)
		checkfunc func(tdir string) error
		args      args
		wantErr   bool
	}{
		{
			name: "success",
			muxfunc: func(mux *http.ServeMux) {
				mux.HandleFunc("/archive", testtarh)
				mux.HandleFunc("/repos/acme/foo/tarball/master", arclinkh)
			},
			checkfunc: func(tdir string) error {
				var files []string
				f, err := os.Open(tdir)
				if err != nil {
					return fmt.Errorf("error opening temp dir: %w", err)
				}
				fi, err := f.Readdir(-1)
				f.Close()
				if err != nil {
					return fmt.Errorf("error reading temp dir: %w", err)
				}
				for _, entry := range fi {
					files = append(files, entry.Name())
				}
				if len(files) != 1 {
					return fmt.Errorf("unexpected number of files (wanted 1): %v", files)
				}
				if files[0] != "foo" {
					return fmt.Errorf("unexpected directory name: %v", files[0])
				}
				if !fi[0].IsDir() {
					return fmt.Errorf("foo should be a directory")
				}
				return nil
			},
			args: args{
				repo: "acme/foo",
				ref:  "master",
			},
		},
		{
			name: "not found",
			muxfunc: func(mux *http.ServeMux) {
				mux.HandleFunc("/archive", testtarh)
				mux.HandleFunc("/repos/acme/foo/tarball/master", arclinkh)
			},
			args: args{
				repo: "acme/somethingelse",
				ref:  "master",
			},
			wantErr: true,
		},
		{
			name: "corrupt archive",
			muxfunc: func(mux *http.ServeMux) {
				mux.HandleFunc("/archive", func(w http.ResponseWriter, r *http.Request) {
					w.Header().Add("Content-Type", "application/octet-stream")
					w.Write([]byte("asdfasdfasdfasdfasdfasdfasdfasdfasdf"))
				})
				mux.HandleFunc("/repos/acme/foo/tarball/master", arclinkh)
			},
			args: args{
				repo: "acme/foo",
				ref:  "master",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mux, _, teardown := testGHClient()
			defer teardown()
			if tt.muxfunc != nil {
				tt.muxfunc(mux)
			}
			gf := &GitHubFetcher{
				c: client,
			}
			tdir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Errorf("error making tempdir: %v", err)
			}
			defer os.RemoveAll(tdir)
			if err := gf.Fetch(context.Background(), tt.args.repo, tt.args.ref, tdir); (err != nil) != tt.wantErr {
				t.Errorf("Fetch() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.checkfunc != nil {
				if err := tt.checkfunc(tdir); err != nil {
					t.Errorf("result check failed: %v", err)
				}
			}
		})
	}
}
