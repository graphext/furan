package jobrunner

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
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
	client    kubernetes.Interface
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
func (kr K8sJobRunner) Run(build models.Build) (*JobWatcher, error) {
	if kr.JobFunc == nil || kr.client == nil {
		return nil, fmt.Errorf("JobFunc is required")
	}
	j := kr.JobFunc(kr.imageInfo, build)
	jo, err := kr.client.BatchV1().Jobs(kr.imageInfo.Namespace).Create(context.Background(), j, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating job: %w", err)
	}
	w, err := kr.client.BatchV1().Jobs(kr.imageInfo.Namespace).Watch(context.Background(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s,metadata.namespace=%s", jo.Name, jo.Namespace),
	})
	jw := &JobWatcher{
		c:            kr.client,
		JobName:      jo.Name,
		JobNamespace: jo.Namespace,
		matchLabels:  jo.Spec.Selector.MatchLabels,
	}
	jw.init(w, 0)
	go jw.start()
	return jw, nil
}

const DefaultJobWatcherTimeout = 30 * time.Minute

// JobWatcher is an object that keeps track of a newly-created k8s Job and
// signals when it succeeds or fails
type JobWatcher struct {
	c                     kubernetes.Interface
	JobName, JobNamespace string
	matchLabels           map[string]string
	timeout               time.Duration
	w                     watch.Interface
	e                     chan error
	d, stop               chan struct{}
}

func (jw *JobWatcher) init(w watch.Interface, timeout time.Duration) {
	jw.w = w
	jw.timeout = timeout
	jw.e = make(chan error)
	jw.d = make(chan struct{})
	jw.stop = make(chan struct{})
}

func (jw *JobWatcher) start() {
	if jw.timeout == 0 {
		jw.timeout = DefaultJobWatcherTimeout
	}
	ticker := time.NewTicker(jw.timeout)
	defer ticker.Stop()
	for {
		select {
		case <-jw.stop:
			return
		case <-ticker.C:
			jw.e <- fmt.Errorf("timeout reached: %v", jw.timeout)
			return
		case e := <-jw.w.ResultChan():
			jw.processEvent(e)
		}
	}
}

func (jw *JobWatcher) processEvent(e watch.Event) {
	if e.Object == nil {
		return
	}
	switch e.Type {
	case watch.Deleted:
		jw.e <- fmt.Errorf("job was deleted")
	case watch.Error:
		jw.e <- fmt.Errorf("error: %v", e.Object)
	case watch.Modified:
		j, ok := e.Object.(*batchv1.Job)
		if !ok {
			jw.e <- fmt.Errorf("watch modified event object is not a job: %T", e.Object)
			return
		}
		if j.Status.Succeeded > 0 {
			jw.d <- struct{}{}
			return
		}
		if j.Status.Failed > 0 {
			jw.e <- fmt.Errorf("%v failed pods", j.Status.Failed)
		}
	}
}

func (jw *JobWatcher) Close() {
	jw.w.Stop()
	if jw.e != nil {
		close(jw.e)
	}
	if jw.d != nil {
		close(jw.d)
	}
	if jw.stop != nil {
		close(jw.stop)
	}
}

func (jw *JobWatcher) Error() chan error {
	return jw.e
}

func (jw *JobWatcher) Done() chan struct{} {
	return jw.d
}

var (
	PodMaxLines = int64(1000)
)

// Logs fetches the logs for pods associated with this job and
// returns a map of pod name to a map of container name to the last
// PodMaxLines of log output for that container.
// Ex:
// logByteSlice := output["pod-name-xyz111"]["app-container"]
func (jw *JobWatcher) Logs() (map[string]map[string][]byte, error) {
	if len(jw.matchLabels) == 0 {
		return nil, fmt.Errorf("empty job match labels")
	}
	selector := []string{}
	for k, v := range jw.matchLabels {
		selector = append(selector, fmt.Sprintf("%v = %v", k, v))
	}
	pl, err := jw.c.CoreV1().Pods(jw.JobNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: strings.Join(selector, ",")})
	if err != nil || pl == nil {
		return nil, fmt.Errorf("error listing pods for selector: %v: %w", selector, err)
	}
	out := make(map[string]map[string][]byte, len(pl.Items))
	for _, p := range pl.Items {
		podlogs, err := jw.podlogs(p)
		if err != nil {
			return nil, fmt.Errorf("error getting logs for pod: %v: %w", p.Name, err)
		}
		out[p.Name] = podlogs
	}
	return out, nil
}

func (jw *JobWatcher) podlogs(pod corev1.Pod) (map[string][]byte, error) {
	out := make(map[string][]byte, len(pod.Status.ContainerStatuses))
	plopts := corev1.PodLogOptions{
		TailLines: &PodMaxLines,
	}
	for _, c := range pod.Status.ContainerStatuses {
		plopts.Container = c.Name
		req := jw.c.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &plopts)
		if req == nil {
			out[c.Name] = []byte{}
			continue
		}
		req.BackOff(nil) // fixes tests with fake client
		logrc, err := req.Stream(context.Background())
		if err != nil {
			return nil, fmt.Errorf("error getting logs: %w", err)
		}
		defer logrc.Close()
		logs, err := ioutil.ReadAll(logrc)
		if err != nil {
			return nil, fmt.Errorf("error reading logs: %w", err)
		}
		out[c.Name] = logs
	}
	return out, nil
}
