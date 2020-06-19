package jobrunner

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/dollarshaveclub/furan/pkg/generated/furanrpc"
	"github.com/dollarshaveclub/furan/pkg/models"
)

func TestK8sJobRunner_Run(t *testing.T) {
	type fields struct {
		client    kubernetes.Interface
		imageInfo ImageInfo
		JobFunc   JobFactoryFunc
	}
	type args struct {
		build models.Build
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		verifyf func(f fields, args args) error
		wantErr bool
	}{
		//
		{
			name: "create job",
			fields: fields{
				client: fake.NewSimpleClientset(),
				imageInfo: ImageInfo{
					Namespace:        "foo",
					PodName:          "foo-zxcvb",
					Image:            "acme/foo:bar",
					ImagePullSecrets: []string{"asdf"},
				},
				JobFunc: func(info ImageInfo, build models.Build) *batchv1.Job {
					j := &batchv1.Job{}
					j.Name = "foo-build-" + build.ID.String()
					j.Labels = map[string]string{"build_id": build.ID.String()}
					return j
				},
			},
			args: args{
				build: models.Build{
					ID:         uuid.Must(uuid.NewV4()),
					Created:    time.Now().UTC(),
					GitHubRepo: "acme/foo",
					GitHubRef:  "master",
					ImageRepo:  "acme/foo",
					Request:    furanrpc.BuildRequest{},
				},
			},
			verifyf: func(f fields, args args) error {
				name := "foo-build-" + args.build.ID.String()
				j, err := f.client.BatchV1().Jobs(f.imageInfo.Namespace).Get(name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if _, ok := j.Labels["build_id"]; !ok {
					return fmt.Errorf("job missing build_id label")
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kr := K8sJobRunner{
				client:    tt.fields.client,
				imageInfo: tt.fields.imageInfo,
				JobFunc:   tt.fields.JobFunc,
			}
			if err := kr.Run(tt.args.build); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.verifyf != nil {
				if err := tt.verifyf(tt.fields, tt.args); err != nil {
					t.Errorf("error in test: %v", err)
				}
			}
		})
	}
}

func TestK8sJobRunner_image(t *testing.T) {
	fpod := func(podname, ns, cname, ips string) *corev1.Pod {
		p := &corev1.Pod{}
		p.Name = podname
		p.Namespace = ns
		p.Spec.Containers = []corev1.Container{
			corev1.Container{
				Name:  cname,
				Image: "foo/furan:master",
			},
		}
		p.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
			corev1.LocalObjectReference{Name: ips},
		}
		return p
	}
	tests := []struct {
		name    string
		podObj  *corev1.Pod
		want    ImageInfo
		wantErr bool
	}{
		{
			name:   "success",
			podObj: fpod("furan", "furan-production", "furan", "asdf1234"),
			want: ImageInfo{
				Namespace:        "furan-production",
				PodName:          "furan",
				Image:            "foo/furan:master",
				ImagePullSecrets: []string{"asdf1234"},
			},
		},
		{
			name:    "no env",
			podObj:  nil,
			wantErr: true,
		},
		{
			name:    "no furan container",
			podObj:  fpod("furan", "furan-production", "something", "asdf1234"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := []runtime.Object{}
			if tt.podObj != nil {
				os.Setenv("POD_NAMESPACE", tt.podObj.Namespace)
				os.Setenv("POD_NAME", tt.podObj.Name)
				objs = append(objs, tt.podObj)
			}
			c := fake.NewSimpleClientset(objs...)
			kr := K8sJobRunner{
				client: c,
			}
			got, err := kr.image()
			if err != nil {
				if !tt.wantErr {
					t.Errorf("image() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if tt.wantErr {
				t.Errorf("expected error")
				return
			}
			if got.Namespace != tt.want.Namespace {
				t.Errorf("wanted namespace %v, got %v", tt.want.Namespace, got.Namespace)
			}
			if got.PodName != tt.want.PodName {
				t.Errorf("wanted podname %v, got %v", tt.want.PodName, got.PodName)
			}
			if got.Image != tt.want.Image {
				t.Errorf("wanted image %v, got %v", tt.want.Image, got.Image)
			}
		})
	}
}
