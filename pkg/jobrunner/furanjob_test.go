package jobrunner

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	batchv1 "k8s.io/api/batch/v1"

	"github.com/dollarshaveclub/furan/pkg/models"
)

func TestFuranJobFunc(t *testing.T) {
	type args struct {
		info  ImageInfo
		build models.Build
	}
	tests := []struct {
		name    string
		args    args
		verifyf func(got *batchv1.Job, b models.Build) error
	}{
		{
			name: "furan job",
			args: args{
				info: ImageInfo{
					Namespace:        "foo",
					PodName:          "furan-xyz",
					Image:            "acme/furan:master",
					ImagePullSecrets: []string{"ips"},
				},
				build: models.Build{
					ID:           uuid.Must(uuid.FromString("f3b1ae8c-c9ac-46f2-8b26-8a1e185e0776")),
					GitHubRepo:   "acme/widgets",
					GitHubRef:    "asdf",
					ImageRepos:   []string{"quay.io/acme/widgets"},
					Tags:         []string{"v1.1.0"},
					CommitSHATag: true,
					Status:       models.BuildStatusNotStarted,
				},
			},
			verifyf: func(got *batchv1.Job, b models.Build) error {
				if got.Namespace != "foo" {
					return fmt.Errorf("bad namespace: %v", got.Namespace)
				}
				if got.Name != "furan-build-"+"acme-widgets-"+b.ID.String() {
					return fmt.Errorf("bad job name: %v", got.Name)
				}
				if i := len(got.Labels); i != 1 {
					return fmt.Errorf("bad job label count: %v", i)
				}
				for _, l := range []string{"build-id"} {
					v, ok := got.Labels[l]
					if !ok {
						return fmt.Errorf("missing label: %v", l)
					}
					switch l {
					case "build-id":
						if v != b.ID.String() {
							return fmt.Errorf("labels: bad build id: %v", v)
						}
					default:
						return fmt.Errorf("unknown label: %v", l)
					}
				}
				b2 := models.Build{}
				if err := json.Unmarshal([]byte(got.Annotations["build-request"]), &b2); err != nil {
					return fmt.Errorf("error unmarshaling build request annotation: %w", err)
				}
				if i := len(got.Spec.Template.Spec.Containers); i != 2 {
					return fmt.Errorf("bad container length: %v", i)
				}
				if img := got.Spec.Template.Spec.Containers[0].Image; img != "acme/furan:master" {
					return fmt.Errorf("bad job image: %v", img)
				}
				args := got.Spec.Template.Spec.Containers[0].Args
				if len(args) < 2 {
					return fmt.Errorf("bad container args length: %v", len(args))
				}
				if larg := args[len(args)-1]; larg != b.ID.String() {
					return fmt.Errorf("bad last argument: %v", larg)
				}
				if i := len(got.Spec.Template.Spec.ImagePullSecrets); i != 1 {
					return fmt.Errorf("bad image pull secrets length: %v", i)
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FuranJobFunc(tt.args.info, tt.args.build)
			if tt.verifyf != nil {
				if err := tt.verifyf(got, tt.args.build); err != nil {
					t.Errorf("error: %v", err)
				}
			}
		})
	}
}
