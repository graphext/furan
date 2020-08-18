package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	grpctrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/google.golang.org/grpc"

	"github.com/dollarshaveclub/furan/pkg/datalayer"
	"github.com/dollarshaveclub/furan/pkg/generated/furanrpc"
	"github.com/dollarshaveclub/furan/pkg/models"
)

type Options struct {
	// TraceSvcName is the service name for APM tracing (optional)
	TraceSvcName string
	// CredentialDecryptionKey is the secretbox key used to decrypt the build github credential
	CredentialDecryptionKey [32]byte
	Cache                   models.CacheOpts
	LogFunc                 func(msg string, args ...interface{})
	JobHandoffTimeout       time.Duration
}

// DefaultCacheOpts is the cache options that will be used if not overridden in Options
var DefaultCacheOpts = models.CacheOpts{
	Type:    models.S3CacheType,
	MaxMode: true,
}

// Server represents an object that responds to gRPC calls
type Server struct {
	DL datalayer.DataLayer
	BM models.BuildManager
	// CFFactory is a function that returns a CodeFetcher. Token should be considered optional.
	CFFactory func(token string) models.CodeFetcher
	Opts      Options
	s         *grpc.Server
}

// Listen starts the RPC listener on addr:port and blocks until the server is signalled to stop.
// Always returns a non-nil error.
func (gr *Server) Listen(addr string, port uint) error {
	if gr.DL == nil {
		return fmt.Errorf("DataLayer is required")
	}
	if gr.CFFactory == nil {
		return fmt.Errorf("CFFactory is required")
	}
	addr = fmt.Sprintf("%v:%v", addr, port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		gr.logf("error starting gRPC listener: %v", err)
		return err
	}
	// TODO Note (mk): We should consider upgrading our go grpc package so that we
	// can take advantage of stream interceptor
	ui := grpctrace.UnaryServerInterceptor(grpctrace.WithServiceName(gr.Opts.TraceSvcName))
	s := grpc.NewServer(grpc.UnaryInterceptor(ui))
	gr.s = s
	furanrpc.RegisterFuranExecutorServer(s, gr)
	gr.logf("gRPC listening on: %v", addr)
	return s.Serve(l)
}

func (gr *Server) logf(msg string, args ...interface{}) {
	if gr.Opts.LogFunc != nil {
		gr.Opts.LogFunc(msg, args...)
	}
}

func (gr *Server) buildevent(id uuid.UUID, msg string, args ...interface{}) {
	if err := gr.DL.AddEvent(context.Background(), id, fmt.Sprintf(msg, args...)); err != nil {
		gr.logf("error adding event to build: %v: %v", id, err)
	}
}

// DefaultJobHandoffTimeout is the default maximum amount of time we will wait for job handoff to occur
// once the build job is running
var DefaultJobHandoffTimeout = 5 * time.Minute

// StartBuild creates a new asynchronous build job
func (gr *Server) StartBuild(ctx context.Context, req *furanrpc.BuildRequest) (*furanrpc.BuildRequestResponse, error) {
	if req.GetBuild().GithubRepo == "" {
		return nil, status.Errorf(codes.InvalidArgument, "github repo is required")
	}
	if req.GetBuild().Ref == "" {
		return nil, status.Errorf(codes.InvalidArgument, "github ref is required")
	}
	if _, err := gr.CFFactory(req.GetBuild().GithubCredential).GetCommitSHA(ctx, req.GetBuild().GithubRepo, req.GetBuild().Ref); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error validating repo and ref: %v", err)
	}
	if len(req.GetPush().GetRegistries()) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "at least one image repo is required")
	}
	irepos := make([]string, len(req.GetPush().GetRegistries()))
	for i := range req.GetPush().GetRegistries() {
		irepos[i] = req.GetPush().GetRegistries()[i].Repo
	}
	reqc := *req
	reqc.GetBuild().GithubCredential = ""
	b := models.Build{
		GitHubRepo:        req.GetBuild().GithubRepo,
		GitHubRef:         req.GetBuild().Ref,
		ImageRepos:        irepos,
		Tags:              req.GetBuild().GetTags(),
		CommitSHATag:      req.GetBuild().TagWithCommitSha,
		DisableBuildCache: req.GetBuild().DisableBuildCache,
		Request:           reqc,
		Status:            models.BuildStatusNotStarted,
	}

	if err := b.EncryptAndSetGitHubCredential([]byte(req.GetBuild().GithubCredential), gr.Opts.CredentialDecryptionKey); err != nil {
		return nil, status.Errorf(codes.Internal, "error encrypting credential: %v", err)
	}

	id, err := gr.DL.CreateBuild(ctx, b)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error creating build in db: %v", err)
	}

	dockerfile := "Dockerfile" // default relative Dockerfile path
	if req.GetBuild().DockerfilePath != "" {
		dockerfile = req.GetBuild().DockerfilePath
	}

	copts := gr.Opts.Cache
	if copts.Type == models.UnknownCacheType {
		copts = DefaultCacheOpts
	}
	if req.GetBuild().DisableBuildCache {
		copts.Type = models.DisabledCacheType
	}

	opts := models.BuildOpts{
		BuildID:                id,
		RelativeDockerfilePath: dockerfile,
		BuildArgs:              req.GetBuild().GetArgs(),
		Cache:                  copts,
	}

	// once everything is validated and we are ready to start job,
	// do the job creation and handoff asynchronously and return id to client
	go func() {
		jht := DefaultJobHandoffTimeout
		if gr.Opts.JobHandoffTimeout != 0 {
			jht = gr.Opts.JobHandoffTimeout
		}
		ctx2, cf := context.WithTimeout(context.Background(), jht)
		defer cf()
		gr.buildevent(id, "creating build job")
		if err := gr.BM.Start(ctx2, opts); err != nil {
			gr.buildevent(id, "error starting build job: %v", err)
			// new context because the error could be caused by context cancellation
			if err2 := gr.DL.SetBuildAsCompleted(context.Background(), id, models.BuildStatusFailure); err2 != nil {
				gr.logf("error setting build as failed: %v: %v", id, err2)
			}
			return
		}
		gr.buildevent(id, "build k8s job running, waiting for handoff")
		if err := gr.DL.ListenForBuildRunning(ctx2, id); err != nil {
			gr.buildevent(id, "error waiting for handoff: %v", err)
			// new context because the error could be caused by context cancellation
			if err2 := gr.DL.SetBuildAsCompleted(context.Background(), id, models.BuildStatusFailure); err2 != nil {
				gr.logf("error setting build as failed: %v: %v", id, err2)
			}
			return
		}
		// "build handoff" means the build k8s job is running and the Furan container has started up and updated
		// the build to Running.
		// The Furan container in the k8s job is now responsible for updating this build.
		// Therefore, this goroutine can now exit
		gr.buildevent(id, "successful build handoff")
	}()

	return &furanrpc.BuildRequestResponse{
		BuildId: id.String(),
	}, nil
}

// GetBuildStatus returns the current status for a build
func (gr *Server) GetBuildStatus(ctx context.Context, req *furanrpc.BuildStatusRequest) (_ *furanrpc.BuildStatusResponse, err error) {
	id, err := uuid.FromString(req.BuildId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid build id: %v", err)
	}
	b, err := gr.DL.GetBuildByID(ctx, id)
	if err != nil {
		if err == datalayer.ErrNotFound {
			return nil, status.Errorf(codes.InvalidArgument, "build not found")
		}
		return nil, status.Errorf(codes.Internal, "error getting build: %v", err)
	}
	var started, completed *furanrpc.Timestamp
	started = &furanrpc.Timestamp{
		Seconds: int64(b.Created.Second()),
		Nanos:   int32(b.Created.Nanosecond()),
	}
	if !b.Completed.IsZero() {
		completed = &furanrpc.Timestamp{
			Seconds: int64(b.Completed.Second()),
			Nanos:   int32(b.Completed.Nanosecond()),
		}
	}
	return &furanrpc.BuildStatusResponse{
		BuildId:      b.ID.String(),
		BuildRequest: &b.Request,
		State:        b.Status.State(),
		Started:      started,
		Completed:    completed,
	}, nil
}

// MonitorBuild streams events from a specified build until completion
func (gr *Server) MonitorBuild(req *furanrpc.BuildStatusRequest, stream furanrpc.FuranExecutor_MonitorBuildServer) (err error) {
	id, err := uuid.FromString(req.BuildId)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid build id: %v", err)
	}
	b, err := gr.DL.GetBuildByID(stream.Context(), id)
	if err != nil {
		if err == datalayer.ErrNotFound {
			return status.Errorf(codes.InvalidArgument, "build not found")
		}
		return status.Errorf(codes.Internal, "error getting build: %v", err)
	}
	// if build is completed, return the last event with status and close connection
	if b.Status.TerminalState() {
		var event string
		if len(b.Events) > 0 {
			event = b.Events[len(b.Events)-1]
		}
		if err := stream.Send(&furanrpc.BuildEvent{
			BuildId:      b.ID.String(),
			Message:      event,
			CurrentState: b.Status.State(),
		}); err != nil {
			gr.logf("error sending build event on RPC stream: %v", err)
		}
		return nil
	}
	events := make(chan string)
	defer close(events)
	ctx, cf := context.WithCancel(stream.Context())
	defer cf()

	// this goroutine sends new build messages on the events channel
	// when the context is cancelled above via defer, this goroutine will exit
	go func() {
		err := gr.DL.ListenForBuildEvents(ctx, b.ID, events)
		if err != nil {
			gr.logf("error listening for build events: %v", err)
		}
	}()

	// this goroutine reads from the events channel and sends on the grpc stream
	// when the events channel is closed above via defer, this goroutine will exit
	go func() {
		for event := range events {
			b, err := gr.DL.GetBuildByID(ctx, b.ID)
			if err != nil {
				gr.logf("error getting build: %v", err)
				return
			}
			if err := stream.Send(&furanrpc.BuildEvent{
				BuildId:      b.ID.String(),
				Message:      event,
				CurrentState: b.Status.State(),
			}); err != nil {
				gr.logf("error sending build event on RPC stream: %v", err)
				return
			}
		}
	}()

	// this blocks until the build is completed
	_, err = gr.DL.ListenForBuildCompleted(ctx, b.ID)
	if err != nil {
		return status.Errorf(codes.Internal, "error listening for build completion: %v", err)
	}
	return nil
}

// CancelBuild requests cancellation for a currently-running build
func (gr *Server) CancelBuild(ctx context.Context, req *furanrpc.BuildCancelRequest) (_ *furanrpc.BuildCancelResponse, err error) {
	id, err := uuid.FromString(req.BuildId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid build id: %v", err)
	}
	b, err := gr.DL.GetBuildByID(ctx, id)
	if err != nil {
		if err == datalayer.ErrNotFound {
			return nil, status.Errorf(codes.InvalidArgument, "build not found")
		}
		return nil, status.Errorf(codes.Internal, "error getting build: %v", err)
	}
	if b.Status.TerminalState() || !b.Running() {
		return nil, status.Errorf(codes.FailedPrecondition, "build state not compatible with cancellation: %v", b.Status)
	}
	if err := gr.DL.CancelBuild(ctx, b.ID); err != nil {
		return nil, status.Errorf(codes.Internal, "error cancelling build: %v", err)
	}
	return &furanrpc.BuildCancelResponse{
		BuildId: id.String(),
	}, nil
}

// Shutdown gracefully stops the  server
func (gr *Server) Shutdown() {
	gr.s.GracefulStop()
}
