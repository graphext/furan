package tagcheck

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestQuayChecker_AllTagsExist(t *testing.T) {
	type args struct {
		tags []string
		repo string
	}
	tests := []struct {
		name    string
		args    args
		tagmap  map[string]struct{} // any tags present will return 200, anything absent returns 404
		want    bool
		want1   []string
		wantErr bool
	}{
		{
			name: "all exist",
			args: args{
				tags: []string{"foo", "bar"},
				repo: "quay.io/foo/bar",
			},
			tagmap: map[string]struct{}{
				"foo": struct{}{},
				"bar": struct{}{},
			},
			want:    true,
			want1:   []string{},
			wantErr: false,
		},
		{
			name: "tag missing",
			args: args{
				tags: []string{"foo", "bar"},
				repo: "quay.io/foo/bar",
			},
			tagmap: map[string]struct{}{
				"foo": struct{}{},
			},
			want:    false,
			want1:   []string{"bar"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// url: http://localhost:xxxx/api/v1/repository/%s/tag/%s/images
				route := strings.ReplaceAll(r.URL.String(), "http://", "")
				rs := strings.Split(route, "/")
				if len(rs) != 9 {
					t.Logf("bad route: %v", rs)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				tag := rs[7]
				if _, ok := tt.tagmap[tag]; ok {
					w.WriteHeader(http.StatusOK)
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer s.Close()
			qc := QuayChecker{
				endpoint: s.URL + "/api/v1/repository/%s/tag/%s/images",
			}
			got, got1, err := qc.AllTagsExist(tt.args.tags, tt.args.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("AllTagsExist() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AllTagsExist() got = %v, want %v", got, tt.want)
			}
			if !cmp.Equal(got1, tt.want1) {
				t.Errorf("AllTagsExist() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestQuayChecker_IsQuay(t *testing.T) {
	type args struct {
		repo string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "quay",
			args: args{
				repo: "quay.io/foo/bar",
			},
			want: true,
		},
		{
			name: "ecr",
			args: args{
				repo: "123456789.dkr.ecr.us-west-2.amazonaws.com/widgets",
			},
			want: false,
		},
		{
			name: "docker hub",
			args: args{
				repo: "foo/bar",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qc := QuayChecker{}
			if got := qc.IsQuay(tt.args.repo); got != tt.want {
				t.Errorf("IsQuay() = %v, want %v", got, tt.want)
			}
		})
	}
}
