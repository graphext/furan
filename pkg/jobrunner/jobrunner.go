package jobrunner

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/dollarshaveclub/furan/pkg/datalayer"
	"github.com/dollarshaveclub/furan/pkg/models"
)

// ImageInfo models information about /this/ currently running Furan pod/container
// These are the values that are injected into build jobs
type ImageInfo struct {
	Namespace, PodName, Image string
	ImagePullSecrets          []string
	RootArgs                  []string // All args prior to the "server" command (secrets setup, etc)
}

// JobFactoryFunc is a function that generates a new image build Job given an ImageInfo
type JobFactoryFunc func(info ImageInfo, build models.Build) *batchv1.Job

type K8sJobRunner struct {
	client    kubernetes.Interface
	dl        datalayer.DataLayer
	imageInfo ImageInfo
	JobFunc   JobFactoryFunc
}

func NewInClusterRunner(dl datalayer.DataLayer) (*K8sJobRunner, error) {
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
		dl:     dl,
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

	var furancidx int
	for i, c := range pod.Spec.Containers {
		if c.Name == "furan" {
			out.Image = c.Image
			furancidx = i
		}
	}
	if out.Image == "" {
		return out, fmt.Errorf("furan container not found in pod")
	}

	// "RootArgs" are all arguments prior to the "server" command
	out.RootArgs = []string{}
	for _, arg := range pod.Spec.Containers[furancidx].Args {
		if arg == "server" {
			break
		}
		out.RootArgs = append(out.RootArgs, arg)
	}
	if fargs := pod.Spec.Containers[furancidx].Args; len(out.RootArgs) == len(fargs) {
		return out, fmt.Errorf("server command not found in furan args: %+v", fargs)
	}

	return out, nil
}

// Run starts a new Furan build job and returns immediately
func (kr K8sJobRunner) Run(build models.Build) (models.Job, error) {
	if kr.JobFunc == nil || kr.client == nil {
		return nil, fmt.Errorf("JobFunc is required")
	}
	j := kr.JobFunc(kr.imageInfo, build)
	jo, err := kr.client.BatchV1().Jobs(kr.imageInfo.Namespace).Create(context.Background(), j, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating job: %w", err)
	}
	kr.dl.AddEvent(context.Background(), build.ID, "job created: "+jo.Name)
	w, err := kr.client.BatchV1().Jobs(kr.imageInfo.Namespace).Watch(context.Background(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s,metadata.namespace=%s", jo.Name, jo.Namespace),
	})
	jw := &JobWatcher{
		c:            kr.client,
		dl:           kr.dl,
		buildID:      build.ID,
		JobName:      jo.Name,
		JobNamespace: jo.Namespace,
		matchLabels:  jo.Spec.Selector.MatchLabels,
	}
	jw.init(w, 0)
	go jw.start()
	return jw, nil
}

const (
	DefaultJobWatcherTimeout = 30 * time.Minute
)

// JobWatcher is an object that keeps track of a newly-created k8s Job and
// signals when it succeeds or fails
type JobWatcher struct {
	c                     kubernetes.Interface
	dl                    datalayer.DataLayer
	buildID               uuid.UUID
	JobName, JobNamespace string
	matchLabels           map[string]string
	timeout               time.Duration
	w                     watch.Interface
	e                     chan error
	r, stop               chan struct{}
}

var _ models.Job = &JobWatcher{}

func (jw *JobWatcher) init(w watch.Interface, timeout time.Duration) {
	jw.w = w
	jw.timeout = timeout
	jw.e = make(chan error)
	jw.r = make(chan struct{})
	jw.stop = make(chan struct{})
}

func (jw *JobWatcher) start() {
	if jw.timeout == 0 {
		jw.timeout = DefaultJobWatcherTimeout
	}
	brunerr := make(chan error, 1)
	running := make(chan struct{})
	ctx, cf := context.WithTimeout(context.Background(), jw.timeout)
	defer cf()
	go func() {
		if jw == nil || jw.dl == nil {
			return
		}
		defer close(running)
		if err := jw.dl.ListenForBuildRunning(ctx, jw.buildID); err != nil {
			select {
			case <-jw.stop:
				return
			default:
			}
			brunerr <- fmt.Errorf("error listening for build running: %w", err)
			return
		}
		jw.dl.AddEvent(ctx, jw.buildID, "job running, handoff successful")
	}()
	// cleanup
	defer func() {
		jw.w.Stop()
		close(jw.e)
		close(jw.r)
	}()
	ticker := time.NewTicker(jw.timeout)
	defer ticker.Stop()
	for {
		select {
		case <-jw.stop: // stop signalled
			return
		case <-ticker.C: // timeout
			jw.e <- fmt.Errorf("timeout reached: %v", jw.timeout)
			return
		case <-running: // job running (handoff completed)
			jw.r <- struct{}{}
			return
		case err := <-brunerr: // listen for handoff error
			jw.e <- err
			return
		case e := <-jw.w.ResultChan(): // job modification events
			if jw.processEvent(e) {
				return
			}
		}
	}
}

// processEvent processes a watch event and returns whether the job watcher should be stopped
func (jw *JobWatcher) processEvent(e watch.Event) bool {
	if e.Object == nil {
		return false
	}
	switch e.Type {
	case watch.Deleted:
		jw.e <- fmt.Errorf("job was deleted")
		return true
	case watch.Error:
		jw.e <- fmt.Errorf("error: %v", e.Object)
		return true
	case watch.Modified:
		j, ok := e.Object.(*batchv1.Job)
		if !ok {
			jw.e <- fmt.Errorf("watch modified event object is not a job: %T", e.Object)
			return true
		}
		jw.dl.AddEvent(context.Background(), jw.buildID, fmt.Sprintf("job event received: %v; status: %+v", e.Type, j.Status))
		if j.Status.Succeeded > 0 {
			jw.dl.AddEvent(context.Background(), jw.buildID, "job marked as succeeded/finished unexpectedly, ending watch")
			jw.r <- struct{}{}
			return true
		}
		if len(j.Status.Conditions) > 0 {
			lc := j.Status.Conditions[len(j.Status.Conditions)-1]
			if lc.Type == batchv1.JobFailed && lc.Reason == "BackoffLimitExceeded" {
				var bo int32
				if j.Spec.BackoffLimit != nil {
					bo = *j.Spec.BackoffLimit
				}
				jw.e <- fmt.Errorf("job backoff limit exceeded (%v), giving up", bo)
				return true
			}
		}
	}
	return false
}

func (jw *JobWatcher) watchJobPod() {

}

func (jw *JobWatcher) Error() chan error {
	return jw.e
}

func (jw *JobWatcher) Running() chan struct{} {
	return jw.r
}

func (jw *JobWatcher) matchLabelsForJob() ([]string, error) {
	if len(jw.matchLabels) == 0 {
		return nil, fmt.Errorf("empty job match labels")
	}
	selector := []string{}
	for k, v := range jw.matchLabels {
		selector = append(selector, fmt.Sprintf("%v = %v", k, v))
	}
	return selector, nil
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
	selector, err := jw.matchLabelsForJob()
	if err != nil {
		return nil, fmt.Errorf("error getting job match selectors: %w", err)
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
