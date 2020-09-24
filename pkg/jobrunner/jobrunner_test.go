package jobrunner

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/dollarshaveclub/furan/pkg/datalayer"
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
					j.Spec.Selector = &metav1.LabelSelector{
						MatchLabels: map[string]string{},
					}
					return j
				},
			},
			args: args{
				build: models.Build{
					ID:         uuid.Must(uuid.NewV4()),
					Created:    time.Now().UTC(),
					GitHubRepo: "acme/foo",
					GitHubRef:  "master",
					ImageRepos: []string{"acme/foo"},
					Request:    furanrpc.BuildRequest{},
				},
			},
			verifyf: func(f fields, args args) error {
				name := "foo-build-" + args.build.ID.String()
				j, err := f.client.BatchV1().Jobs(f.imageInfo.Namespace).Get(context.Background(), name, metav1.GetOptions{})
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
				dl:        &datalayer.FakeDataLayer{},
				imageInfo: tt.fields.imageInfo,
				JobFunc:   tt.fields.JobFunc,
			}
			if _, err := kr.Run(tt.args.build); (err != nil) != tt.wantErr {
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
	fpod := func(podname, ns, cname, ips string, omitserver bool) *corev1.Pod {
		p := &corev1.Pod{}
		p.Name = podname
		p.Namespace = ns
		args := []string{
			"--foo=bar",
			"--baz=123",
		}
		if !omitserver {
			args = append(args, "server", "--asdf=zxcv")
		}
		p.Spec.Containers = []corev1.Container{
			corev1.Container{
				Name:  cname,
				Image: "foo/furan:master",
				Args:  args,
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
			podObj: fpod("furan", "furan-production", "furan", "asdf1234", false),
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
			podObj:  fpod("furan", "furan-production", "something", "asdf1234", false),
			wantErr: true,
		},
		{
			name:    "no server command",
			podObj:  fpod("furan", "furan-production", "furan", "asdf1234", true),
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

func TestJobWatcher(t *testing.T) {
	stests := []struct {
		name  string
		tfunc func(t *testing.T)
	}{
		{
			name: "success",
			tfunc: func(t *testing.T) {
				fw := watch.NewFake()
				dl := &datalayer.FakeDataLayer{}
				id, err := dl.CreateBuild(context.Background(), models.Build{
					Status: models.BuildStatusNotStarted,
				})
				if err != nil {
					t.Fatalf("error creating build: %v", err)
				}
				jw := &JobWatcher{
					buildID: id,
					dl:      dl,
				}
				jw.init(fw, 1*time.Second)
				sent := make(chan struct{})
				go func() {
					close(sent)
					fw.Modify(&batchv1.Job{
						Status: batchv1.JobStatus{
							Succeeded: 1,
						},
					})
				}()
				go func() {
					<-sent
					jw.start()
				}()
				select {
				case err := <-jw.Error():
					t.Errorf("unexpected error: %v", err)
				case <-jw.Running():
					t.Logf("success")
				}
			},
		},
		{
			name: "error",
			tfunc: func(t *testing.T) {
				fw := watch.NewFake()
				jw := &JobWatcher{
					dl: &datalayer.FakeDataLayer{},
				}
				jw.init(fw, 1*time.Second)
				go jw.start()
				go func() {
					fw.Error(&batchv1.Job{
						Status: batchv1.JobStatus{
							Failed: 1,
						},
					})
				}()
				select {
				case <-jw.Error():
					t.Logf("expected error")
				case <-jw.Running():
					t.Errorf("should not have succeeded")
				}
			},
		},
		{
			name: "timeout",
			tfunc: func(t *testing.T) {
				fw := watch.NewFake()
				dl := &datalayer.FakeDataLayer{}
				id, _ := dl.CreateBuild(context.Background(), models.Build{})
				jw := &JobWatcher{
					dl:      dl,
					buildID: id,
				}
				jw.init(fw, 5*time.Millisecond)
				go jw.start()
				select {
				case err := <-jw.Error():
					t.Logf("expected timeout error: %v", err)
				case <-jw.Running():
					t.Errorf("should not have succeeded")
				}
			},
		},
		// this test needs this PR to be released in client-go:
		// https://github.com/kubernetes/kubernetes/pull/91485
		{
			name: "logs",
			tfunc: func(t *testing.T) {
				fw := watch.NewFake()
				p := &corev1.Pod{}
				p.Name = "foo-asdf1234"
				p.Namespace = "foo"
				p.Labels = map[string]string{"foo": "bar"}
				p.Status = corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						corev1.ContainerStatus{
							Name: "foo-app-container",
						},
					},
				}
				dl := &datalayer.FakeDataLayer{}
				id, _ := dl.CreateBuild(context.Background(), models.Build{})
				fc := fake.NewSimpleClientset(p)
				jw := &JobWatcher{
					c:            fc,
					dl:           dl,
					buildID:      id,
					matchLabels:  p.Labels,
					JobName:      "foo-job",
					JobNamespace: "foo",
				}
				jw.init(fw, 1*time.Second)
				go jw.start()
				go func() {
					time.Sleep(10 * time.Millisecond)
					fw.Modify(&batchv1.Job{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "foo-job",
							Namespace: "foo",
						},
						Spec: batchv1.JobSpec{
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"foo": "bar",
								},
							},
						},
						Status: batchv1.JobStatus{
							Succeeded: 1,
						},
					})
				}()
				select {
				case err := <-jw.Error():
					t.Errorf("unexpected error: %v", err)
				case <-jw.Running():
					t.Logf("success")
				}
				logs, err := jw.Logs()
				if err != nil {
					t.Errorf("error getting logs: %v", err)
				}
				if len(logs) != 1 {
					t.Errorf("unexpected logs length: %v", len(logs))
				}
				t.Logf("logs: %+v", string(logs["foo-asdf1234"]["foo-app-container"]))
			},
		},
	}
	for _, tt := range stests {
		t.Run(tt.name, tt.tfunc)
	}
}
