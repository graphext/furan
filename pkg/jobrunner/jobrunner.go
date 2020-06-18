package jobrunner

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/dollarshaveclub/furan/pkg/models"
)

// ImageInfo models information about /this/ currently running Furan pod/container
type ImageInfo struct {
	Namespace, PodName, Image string
	ImagePullSecrets          []string
}

// JobFactoryFunc is a function that generates a new image build Job given an ImageInfo
type JobFactoryFunc func(info ImageInfo, build models.Build) *batchv1.Job

type K8sJobRunner struct {
	client    *kubernetes.Clientset
	imageInfo ImageInfo
	JobFunc   JobFactoryFunc
}

func NewInClusterRunner() (*K8sJobRunner, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("error getting in-cluster config: %w", err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error getting client: %w", err)
	}
	kr := &K8sJobRunner{
		client: client,
	}
	iinfo, err := kr.image()
	if err != nil {
		return nil, fmt.Errorf("error getting image details: %w", err)
	}
	kr.imageInfo = iinfo
	return kr, nil
}

// namespace returns the namespace we're currently running in, or "default"
func (kr K8sJobRunner) namespace() string {
	// using the downward API
	if ns, ok := os.LookupEnv("POD_NAMESPACE"); ok {
		return ns
	}

	// Fall back to the namespace associated with the service account token, if available
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}

	return "default"
}

// image returns the image info for the pod/container we are currently running in, or error
func (kr K8sJobRunner) image() (ImageInfo, error) {
	// Per https://github.com/kubernetes/kubernetes/issues/19475, it's not currently possible to use downward API to
	// get the image tag, so we have to interrogate the k8s API

	out := ImageInfo{}
	out.Namespace = kr.namespace()

	podn, ok := os.LookupEnv("POD_NAME")
	if !ok {
		return out, fmt.Errorf("POD_NAME not found")
	}

	out.PodName = podn

	pod, err := kr.client.CoreV1().Pods(out.Namespace).Get(context.Background(), out.PodName, metav1.GetOptions{})
	if err != nil {
		return out, fmt.Errorf("error getting pod: %w", err)
	}

	for _, ips := range pod.Spec.ImagePullSecrets {
		out.ImagePullSecrets = append(out.ImagePullSecrets, ips.Name)
	}

	for _, c := range pod.Spec.Containers {
		if c.Name == "furan" {
			out.Image = c.Image
			return out, nil
		}
	}

	return out, fmt.Errorf("furan container not found in pod")
}

// Run starts a new Furan build job and returns immediately
func (kr K8sJobRunner) Run(build models.Build) error {
	if kr.JobFunc == nil || kr.client == nil {
		return fmt.Errorf("JobFunc is required")
	}
	j := kr.JobFunc(kr.imageInfo, build)
	_, err := kr.client.BatchV1().Jobs(kr.imageInfo.Namespace).Create(context.Background(), j, metav1.CreateOptions{})
	return err
}
