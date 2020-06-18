package jobrunner

import (
	"encoding/json"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dollarshaveclub/furan/pkg/models"
)

var (
	jobParallelism           = int32(1)
	jobCompletions           = int32(1)
	jobBackoffLimit          = int32(3)
	jobActiveDeadlineSeconds = int64(60 * 60) // 1 hour
)

// furanjob models a Job to execute a single image build/push
// It utilizes BuildKit listening on a UNIX socket, shared with the Furan container by an emptyDir volume
// The Furan container (sidecar, in a sense) executes the build/push via the BuildKit gRPC API, records progress in the
// database and exits cleanly when finished
var furanjob = batchv1.Job{
	ObjectMeta: metav1.ObjectMeta{
		Name:        "",
		Namespace:   "",
		Labels:      nil,
		Annotations: nil,
	},
	Spec: batchv1.JobSpec{
		Parallelism:           &jobParallelism,
		Completions:           &jobCompletions,
		ActiveDeadlineSeconds: &jobActiveDeadlineSeconds,
		BackoffLimit:          &jobBackoffLimit,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				// Metadata is set below
				Name: "",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					corev1.Container{
						Name:            "furan",
						Image:           "",
						ImagePullPolicy: "IfNotPresent",
						Args: []string{
							"runbuild",
							// "--build-id <id>" is injected below
						},
						VolumeMounts: []corev1.VolumeMount{
							corev1.VolumeMount{
								Name:      "certs",
								ReadOnly:  true,
								MountPath: "/certs",
							},
							corev1.VolumeMount{
								Name:      "bksocket",
								MountPath: "/run/buildkit",
							},
						},
					},
					corev1.Container{
						Name:            "buildkitd",
						Image:           "moby/buildkit:v0.7.1",
						ImagePullPolicy: "IfNotPresent",
						Args: []string{
							"--addr",
							"unix:///run/buildkit/buildkitd.sock",
							"--tlscacert",
							"/certs/ca.pem",
							"--tlscert",
							"/certs/cert.pem",
							"--tlskey",
							"/certs/key.pem",
						},
						VolumeMounts: []corev1.VolumeMount{
							corev1.VolumeMount{
								Name:      "certs",
								ReadOnly:  true,
								MountPath: "/certs",
							},
							corev1.VolumeMount{
								Name:      "bksocket",
								MountPath: "/run/buildkit",
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					corev1.Volume{
						Name: "bksocket",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{
								Medium: corev1.StorageMediumMemory,
							},
						},
					},
					corev1.Volume{
						Name: "certs",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "buildkit-daemon-certs", // managed by Furan chart
							},
						},
					},
				},
				ImagePullSecrets: nil,
			},
		},
	},
}

// FuranJobFunc is a JobFactoryFunc that generates a Kubernetes Job to execute a build
func FuranJobFunc(info ImageInfo, build models.Build) *batchv1.Job {
	j := furanjob
	j.Namespace = info.Namespace
	j.Name = "furan-build-" + build.ID.String()
	j.Labels = map[string]string{
		"build-id":    build.ID.String(),
		"source-repo": build.GitHubRepo,
		"source-ref":  build.GitHubRef,
		"dest-repo":   build.ImageRepo,
		"image-tags":  strings.Join(build.Tags, ","),
	}
	reqj, _ := json.Marshal(build.Request) // ignore error
	j.Annotations = map[string]string{
		"build-request": string(reqj), // if reqj == nil, this will be an empty string
	}
	j.Spec.Template.Spec.Containers[0].Image = info.Image
	j.Spec.Template.Spec.Containers[0].Args = append(
		j.Spec.Template.Spec.Containers[0].Args,
		"--build-id",
		build.ID.String())
	for _, ips := range info.ImagePullSecrets {
		j.Spec.Template.Spec.ImagePullSecrets = append(
			j.Spec.Template.Spec.ImagePullSecrets,
			corev1.LocalObjectReference{Name: ips})
	}
	return &j
}

var _ JobFactoryFunc = FuranJobFunc
