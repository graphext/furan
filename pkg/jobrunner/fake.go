package jobrunner

import "github.com/dollarshaveclub/furan/pkg/models"

type FakeJob struct {
	ErrorChan   chan error
	RunningChan chan struct{}
	LogContent  map[string]map[string][]byte
}

var _ models.Job = &FakeJob{}

func NewFakeJob(logs map[string]map[string][]byte) *FakeJob {
	return &FakeJob{
		ErrorChan:   make(chan error),
		RunningChan: make(chan struct{}),
		LogContent:  logs,
	}
}

func (fj *FakeJob) Close() {}
func (fj *FakeJob) Error() chan error {
	return fj.ErrorChan
}
func (fj *FakeJob) Running() chan struct{} {
	return fj.RunningChan
}
func (fj *FakeJob) Logs() (map[string]map[string][]byte, error) {
	return fj.LogContent, nil
}

type FakeJobRunner struct {
	RunFunc func(build models.Build) (models.Job, error)
}

func (fj *FakeJobRunner) Run(build models.Build) (models.Job, error) {
	if fj.RunFunc != nil {
		return fj.RunFunc(build)
	}
	return NewFakeJob(nil), nil
}

var _ models.JobRunner = &FakeJobRunner{}
