package builder

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/dollarshaveclub/furan/pkg/buildkit"
	"github.com/dollarshaveclub/furan/pkg/datalayer"
	"github.com/dollarshaveclub/furan/pkg/generated/furanrpc"
	"github.com/dollarshaveclub/furan/pkg/models"
	"github.com/dollarshaveclub/furan/pkg/tagcheck"
)

type fakeJob struct {
	ErrorChan   chan error
	RunningChan chan struct{}
	LogContent  map[string]map[string][]byte
}

var _ models.Job = &fakeJob{}

func newFakeJob(logs map[string]map[string][]byte) *fakeJob {
	return &fakeJob{
		ErrorChan:   make(chan error),
		RunningChan: make(chan struct{}),
		LogContent:  logs,
	}
}

func (fj *fakeJob) Close() {}
func (fj *fakeJob) Error() chan error {
	return fj.ErrorChan
}
func (fj *fakeJob) Running() chan struct{} {
	return fj.RunningChan
}
func (fj *fakeJob) Logs() (map[string]map[string][]byte, error) {
	return fj.LogContent, nil
}

type fakeJRunner struct {
	RunFunc func(build models.Build) (models.Job, error)
}

func (fj *fakeJRunner) Run(build models.Build) (models.Job, error) {
	if fj.RunFunc != nil {
		return fj.RunFunc(build)
	}
	return newFakeJob(nil), nil
}

type fakeFetcher struct {
	FetchFunc        func(ctx context.Context, repo string, ref string, destinationPath string) error
	GetCommitSHAFunc func(ctx context.Context, repo string, ref string) (string, error)
}

func (ff *fakeFetcher) Fetch(ctx context.Context, repo string, ref string, destinationPath string) error {
	if ff.FetchFunc != nil {
		return ff.FetchFunc(ctx, repo, ref, destinationPath)
	}
	return nil
}

func (ff *fakeFetcher) GetCommitSHA(ctx context.Context, repo string, ref string) (string, error) {
	if ff.GetCommitSHAFunc != nil {
		return ff.GetCommitSHAFunc(ctx, repo, ref)
	}
	return "", nil
}

func TestManager_Start(t *testing.T) {
	type fields struct {
		BRunner        BuildRunner
		TCheck         TagChecker
		FetcherFactory func(token string) models.CodeFetcher
	}
	type args struct {
		opts models.BuildOpts
	}
	tests := []struct {
		name                          string
		fields                        fields
		args                          args
		ctxCancel, joberr, jobrunning bool
		wantErr                       bool
	}{
		{
			name: "success",
			fields: fields{
				BRunner: &buildkit.BuildSolver{},
				TCheck:  &tagcheck.Checker{},
				FetcherFactory: func(token string) models.CodeFetcher {
					return &fakeFetcher{}
				},
			},
			args: args{
				opts: models.BuildOpts{},
			},
			jobrunning: true,
		},
		{
			name: "job error",
			fields: fields{
				BRunner: &buildkit.BuildSolver{},
				TCheck:  &tagcheck.Checker{},
				FetcherFactory: func(token string) models.CodeFetcher {
					return &fakeFetcher{}
				},
			},
			args: args{
				opts: models.BuildOpts{},
			},
			joberr:  true,
			wantErr: true,
		},
		{
			name: "context cancelled",
			fields: fields{
				BRunner: &buildkit.BuildSolver{},
				TCheck:  &tagcheck.Checker{},
				FetcherFactory: func(token string) models.CodeFetcher {
					return &fakeFetcher{}
				},
			},
			args: args{
				opts: models.BuildOpts{},
			},
			ctxCancel: true,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.ctxCancel && !tt.joberr && !tt.jobrunning {
				t.Fatalf("at least one outcome is required for test")
			}
			ctx := context.Background()
			dl := &datalayer.FakeDataLayer{}
			id, _ := dl.CreateBuild(ctx, models.Build{
				GitHubRepo: "acme/foo",
				GitHubRef:  "master",
				ImageRepos: []string{"acme/foo"},
			})
			tt.args.opts.BuildID = id
			fjr := &fakeJRunner{
				RunFunc: func(build models.Build) (models.Job, error) {
					fj := newFakeJob(nil)
					if tt.jobrunning {
						go func() {
							time.Sleep(10 * time.Millisecond)
							close(fj.RunningChan)
						}()
					}
					if tt.joberr {
						go func() {
							time.Sleep(10 * time.Millisecond)
							fj.ErrorChan <- fmt.Errorf("error in job")
						}()
					}
					return fj, nil
				},
			}
			m := &Manager{
				JRunner:        fjr,
				BRunner:        tt.fields.BRunner,
				TCheck:         tt.fields.TCheck,
				FetcherFactory: tt.fields.FetcherFactory,
				DL:             dl,
			}
			if tt.ctxCancel {
				ctx2, cf := context.WithCancel(ctx)
				go func() { time.Sleep(10 * time.Millisecond); cf() }()
				ctx = ctx2
			}
			if err := m.Start(ctx, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManager_Monitor(t *testing.T) {
	tests := []struct {
		name        string
		build       *models.Build
		buildStatus models.BuildStatus
		msgs        []string
		wantErr     bool
		wantMsgs    int
	}{
		{
			name: "success",
			build: &models.Build{
				GitHubRepo: "acme/foo",
				GitHubRef:  "bar",
				ImageRepos: []string{"acme/foo"},
			},
			buildStatus: models.BuildStatusSuccess,
			msgs:        []string{"foo", "bar", "baz"},
			wantErr:     false,
			wantMsgs:    4,
		},
		{
			name: "failure",
			build: &models.Build{
				GitHubRepo: "acme/foo",
				GitHubRef:  "bar",
				ImageRepos: []string{"acme/foo"},
			},
			buildStatus: models.BuildStatusFailure,
			msgs:        []string{"foo", "bar", "baz"},
			wantErr:     false,
			wantMsgs:    4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			dl := &datalayer.FakeDataLayer{}
			var id uuid.UUID
			if tt.build != nil {
				id, _ = dl.CreateBuild(ctx, *tt.build)
				dl.SetBuildAsRunning(ctx, id)
			}
			m := &Manager{
				DL:      dl,
				JRunner: &fakeJRunner{},
				BRunner: &buildkit.BuildSolver{},
				TCheck:  &tagcheck.Checker{},
				FetcherFactory: func(token string) models.CodeFetcher {
					return &fakeFetcher{}
				},
			}
			start := make(chan struct{})
			done := make(chan struct{})
			done2 := make(chan struct{})
			msgs := make(chan string)
			var i int
			go func() {
				for _ = range msgs {
					i++
				}
				close(done2)
			}()
			go func() {
				<-start
				for _, msg := range tt.msgs {
					if err := dl.AddEvent(ctx, id, msg); err != nil {
						t.Logf("error adding event: %v: %v", msg, err)
					}
				}
				if err := dl.SetBuildAsCompleted(ctx, id, tt.buildStatus); err != nil {
					t.Logf("error setting build as completed: %v", err)
				}
				close(done)
			}()
			close(start)
			if err := m.Monitor(ctx, id, msgs); (err != nil) != tt.wantErr {
				t.Errorf("Monitor() error = %v, wantErr %v", err, tt.wantErr)
			}
			close(msgs)
			<-done
			<-done2
			if i != tt.wantMsgs {
				t.Errorf("wanted %d msgs but got %d", tt.wantMsgs, i)
			}
		})
	}
}

func TestManager_Cancel(t *testing.T) {
	tests := []struct {
		name    string
		wait    bool
		build   *models.Build
		wantErr bool
	}{
		{
			name: "wait",
			wait: true,
			build: &models.Build{
				GitHubRepo: "acme/foo",
				GitHubRef:  "bar",
				ImageRepos: []string{"acme/foo"},
			},
			wantErr: false,
		},
		{
			name: "no wait",
			wait: false,
			build: &models.Build{
				GitHubRepo: "acme/foo",
				GitHubRef:  "bar",
				ImageRepos: []string{"acme/foo"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			dl := &datalayer.FakeDataLayer{}
			var id uuid.UUID
			if tt.build != nil {
				id, _ = dl.CreateBuild(ctx, *tt.build)
				dl.SetBuildAsRunning(ctx, id)
			}
			m := &Manager{
				DL:      dl,
				JRunner: &fakeJRunner{},
				BRunner: &buildkit.BuildSolver{},
				TCheck:  &tagcheck.Checker{},
				FetcherFactory: func(token string) models.CodeFetcher {
					return &fakeFetcher{}
				},
			}
			if tt.wait {
				go func() {
					for {
						id, _ := dl.GetBuildByID(ctx, id)
						if id.Status == models.BuildStatusCancelRequested {
							break
						}
						time.Sleep(1 * time.Millisecond)
					}
					dl.SetBuildAsCompleted(ctx, id, models.BuildStatusCancelled)
				}()
			}
			if err := m.Cancel(ctx, id, tt.wait); (err != nil) != tt.wantErr {
				t.Errorf("Cancel() error = %v, wantErr %v", err, tt.wantErr)
			}
			wantStatus := models.BuildStatusCancelRequested
			b, _ := dl.GetBuildByID(ctx, id)
			if tt.wait {
				wantStatus = models.BuildStatusCancelled
			}
			if b.Status != wantStatus {
				t.Errorf("bad status: %v (wanted %v)", b.Status, wantStatus)
			}
		})
	}
}

type fakeChecker struct {
	AllTagsExistFunc func(tags []string, repo string) (bool, []string, error)
}

func (fc *fakeChecker) AllTagsExist(tags []string, repo string) (bool, []string, error) {
	if fc.AllTagsExistFunc != nil {
		return fc.AllTagsExistFunc(tags, repo)
	}
	return false, nil, nil
}

var _ TagChecker = &fakeChecker{}

type fakeRunner struct {
	BuildFunc func(ctx context.Context, opts models.BuildOpts) error
}

func (fr *fakeRunner) Build(ctx context.Context, opts models.BuildOpts) error {
	if fr.BuildFunc != nil {
		return fr.BuildFunc(ctx, opts)
	}
	return nil
}

var _ BuildRunner = &fakeRunner{}

func TestManager_Run(t *testing.T) {
	tests := []struct {
		name             string
		decrypterr       bool
		commitSHA        string
		commitSHAerr     error
		allTagsExistFunc func(tags []string, repo string) (bool, []string, error)
		imagerepos       []string
		fetcherr         error
		cancel           bool
		builderr         error
		wantErr          bool
		verifyfunc       func(dl datalayer.DataLayer, id uuid.UUID) error
	}{
		{
			name:      "success",
			commitSHA: "asdf",
			allTagsExistFunc: func(tags []string, repo string) (bool, []string, error) {
				return false, tags, nil
			},
			verifyfunc: func(dl datalayer.DataLayer, id uuid.UUID) error {
				b, _ := dl.GetBuildByID(context.Background(), id)
				if b.Status != models.BuildStatusSuccess {
					return fmt.Errorf("bad status: %v (wanted Success)", b.Status)
				}
				return nil
			},
		},
		{
			name:      "cancelled",
			commitSHA: "asdf",
			allTagsExistFunc: func(tags []string, repo string) (bool, []string, error) {
				return false, tags, nil
			},
			cancel:  true,
			wantErr: true,
			verifyfunc: func(dl datalayer.DataLayer, id uuid.UUID) error {
				b, _ := dl.GetBuildByID(context.Background(), id)
				if b.Status != models.BuildStatusCancelled {
					return fmt.Errorf("bad status: %v (wanted Cancelled)", b.Status)
				}
				return nil
			},
		},
		{
			name:      "skipped",
			commitSHA: "asdf",
			allTagsExistFunc: func(tags []string, repo string) (bool, []string, error) {
				return true, []string{}, nil
			},
			verifyfunc: func(dl datalayer.DataLayer, id uuid.UUID) error {
				b, _ := dl.GetBuildByID(context.Background(), id)
				if b.Status != models.BuildStatusSkipped {
					return fmt.Errorf("bad status: %v (wanted Skipped)", b.Status)
				}
				return nil
			},
		},
		{
			name:       "token decrypt error",
			decrypterr: true,
			commitSHA:  "asdf",
			wantErr:    true,
			verifyfunc: func(dl datalayer.DataLayer, id uuid.UUID) error {
				b, _ := dl.GetBuildByID(context.Background(), id)
				if b.Status != models.BuildStatusNotStarted {
					return fmt.Errorf("bad status: %v (wanted NotStarted)", b.Status)
				}
				return nil
			},
		},
		{
			name:         "commit sha error",
			commitSHAerr: fmt.Errorf("something happened"),
			wantErr:      true,
			verifyfunc: func(dl datalayer.DataLayer, id uuid.UUID) error {
				b, _ := dl.GetBuildByID(context.Background(), id)
				if b.Status != models.BuildStatusNotStarted {
					return fmt.Errorf("bad status: %v (wanted NotStarted)", b.Status)
				}
				return nil
			},
		},
		{
			name:      "fetch error",
			commitSHA: "asdf",
			allTagsExistFunc: func(tags []string, repo string) (bool, []string, error) {
				return false, tags, nil
			},
			fetcherr: fmt.Errorf("something happened"),
			wantErr:  true,
			verifyfunc: func(dl datalayer.DataLayer, id uuid.UUID) error {
				b, _ := dl.GetBuildByID(context.Background(), id)
				if b.Status != models.BuildStatusNotStarted {
					return fmt.Errorf("bad status: %v (wanted NotStarted)", b.Status)
				}
				return nil
			},
		},
		{
			name:      "build error",
			commitSHA: "asdf",
			allTagsExistFunc: func(tags []string, repo string) (bool, []string, error) {
				return false, tags, nil
			},
			builderr: fmt.Errorf("something happened"),
			wantErr:  true,
			verifyfunc: func(dl datalayer.DataLayer, id uuid.UUID) error {
				b, _ := dl.GetBuildByID(context.Background(), id)
				if b.Status != models.BuildStatusFailure {
					return fmt.Errorf("bad status: %v (wanted Failure)", b.Status)
				}
				return nil
			},
		},
		{
			name:      "tagcheck error",
			commitSHA: "asdf",
			allTagsExistFunc: func(tags []string, repo string) (bool, []string, error) {
				return false, tags, fmt.Errorf("something happened")
			},
			wantErr: true,
			verifyfunc: func(dl datalayer.DataLayer, id uuid.UUID) error {
				b, _ := dl.GetBuildByID(context.Background(), id)
				if b.Status != models.BuildStatusNotStarted {
					return fmt.Errorf("bad status: %v (wanted NotStarted)", b.Status)
				}
				return nil
			},
		},
		{
			name:      "multi repo tagcheck",
			commitSHA: "asdf",
			allTagsExistFunc: func(tags []string, repo string) (bool, []string, error) {
				if repo == "acme/foo" {
					return true, []string{}, nil
				}
				return false, tags, nil
			},
			imagerepos: []string{"acme/foo", "acme/bar"},
			verifyfunc: func(dl datalayer.DataLayer, id uuid.UUID) error {
				b, _ := dl.GetBuildByID(context.Background(), id)
				if b.Status != models.BuildStatusSuccess {
					return fmt.Errorf("bad status: %v (wanted Success)", b.Status)
				}
				return nil
			},
		},
		{
			name:      "multi repo tagcheck - skip build",
			commitSHA: "asdf",
			allTagsExistFunc: func(tags []string, repo string) (bool, []string, error) {
				return true, []string{}, nil
			},
			imagerepos: []string{"acme/foo", "acme/bar"},
			verifyfunc: func(dl datalayer.DataLayer, id uuid.UUID) error {
				b, _ := dl.GetBuildByID(context.Background(), id)
				if b.Status != models.BuildStatusSkipped {
					return fmt.Errorf("bad status: %v (wanted Skipped)", b.Status)
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			dl := &datalayer.FakeDataLayer{}
			if tt.imagerepos == nil {
				tt.imagerepos = []string{"acme/foo"}
			}
			b := models.Build{
				Status:     models.BuildStatusNotStarted,
				GitHubRepo: "acme/foo",
				GitHubRef:  "bar",
				ImageRepos: tt.imagerepos,
				Request: furanrpc.BuildRequest{
					SkipIfExists: true,
				},
			}

			ghtoken := "asdf1234"

			keybs := make([]byte, 32)
			if n, err := rand.Read(keybs); err != nil || n != len(keybs) {
				t.Errorf("error getting random key: %v", err)
			}
			var key [32]byte
			copy(key[:], keybs)
			if err := b.EncryptAndSetGitHubCredential([]byte(ghtoken), key); err != nil {
				t.Errorf("error encrypting token: %v", err)
			}

			id, _ := dl.CreateBuild(ctx, b)

			ff := func(token string) models.CodeFetcher {
				if token != ghtoken {
					t.Errorf("bad github token: %v (wanted %v)", token, ghtoken)
				}
				return &fakeFetcher{
					GetCommitSHAFunc: func(ctx context.Context, repo string, ref string) (string, error) {
						if repo != b.GitHubRepo {
							t.Errorf("bad repo: %v (wanted %v)", repo, b.GitHubRepo)
						}
						if ref != b.GitHubRef {
							t.Errorf("bad ref: %v (wanted %v)", ref, b.GitHubRef)
						}
						return tt.commitSHA, tt.commitSHAerr
					},
					FetchFunc: func(ctx context.Context, repo string, ref string, destinationPath string) error {
						if repo != b.GitHubRepo {
							t.Errorf("bad repo: %v (wanted %v)", repo, b.GitHubRepo)
						}
						if ref != b.GitHubRef {
							t.Errorf("bad ref: %v (wanted %v)", ref, b.GitHubRef)
						}
						return tt.fetcherr
					},
				}
			}

			buildEntered := make(chan struct{})

			fr := &fakeRunner{
				BuildFunc: func(ctx context.Context, opts models.BuildOpts) error {
					close(buildEntered)
					if opts.CommitSHA != tt.commitSHA {
						t.Errorf("bad commit sha in opts: %v (wanted %v)", opts.CommitSHA, tt.commitSHA)
					}
					if opts.BuildID != id {
						t.Errorf("bad build id in opts: %v (wanted %v)", opts.BuildID, id)
					}
					if tt.cancel {
						<-ctx.Done()
						return fmt.Errorf("context was cancelled")
					}
					return tt.builderr
				},
			}

			m := &Manager{
				DL:             dl,
				BRunner:        fr,
				TCheck:         &fakeChecker{AllTagsExistFunc: tt.allTagsExistFunc},
				FetcherFactory: ff,
				GitHubTokenKey: key,
				JRunner:        &fakeJRunner{},
			}

			if tt.decrypterr {
				m.GitHubTokenKey = [32]byte{}
			}

			if tt.cancel {
				go func() {
					<-buildEntered
					// make sure we have at least one listener waiting for cancellation
					for {
						if dl.CancellationListeners() > 0 {
							break
						}
						time.Sleep(1 * time.Millisecond)
					}
					dl.CancelBuild(ctx, id)
				}()
			}

			if err := m.Run(ctx, models.BuildOpts{BuildID: id}); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.verifyfunc != nil {
				if err := tt.verifyfunc(dl, id); err != nil {
					t.Errorf("verify failed: %v", err)
				}
			}
		})
	}
}
