package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/dollarshaveclub/furan/v2/pkg/client"
	"github.com/dollarshaveclub/furan/v2/pkg/generated/furanrpc"
	"github.com/dollarshaveclub/furan/v2/pkg/models"
)

var monitorBuild bool
var buildArgs []string
var cachetype string
var cachemaxmode bool
var rpctimeout, buildtimeout time.Duration

// triggerCmd represents the trigger command
var triggerCmd = &cobra.Command{
	Use:   "trigger",
	Short: "Start a build on a remote Furan server",
	Long:  `Trigger and then monitor a build on a remote Furan server`,
	RunE:  trigger,
}

var triggerBuildRequest = furanrpc.BuildRequest{
	Build: &furanrpc.BuildDefinition{Resources: &furanrpc.BuildResources{}},
	Push:  &furanrpc.PushDefinition{},
}

var imagerepos, tags []string

var clientops client.Options

func setclientflags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&clientops.Address, "remote-host", "", "Remote Furan server with gRPC port (eg: furan.me.com:4001)")
	cmd.PersistentFlags().StringVar(&clientops.APIKey, "api-key", "", "API key")
	cmd.PersistentFlags().BoolVar(&clientops.TLSInsecureSkipVerify, "tls-skip-verify", false, "Disable TLS certificate verification for RPC calls (INSECURE)")
}

func init() {
	setclientflags(triggerCmd)
	triggerCmd.PersistentFlags().StringVar(&triggerBuildRequest.Build.GithubCredential, "github-token", os.Getenv("GITHUB_TOKEN"), "github token")
	triggerCmd.PersistentFlags().StringVar(&triggerBuildRequest.Build.GithubRepo, "github-repo", "", "source github repo")
	triggerCmd.PersistentFlags().StringVar(&triggerBuildRequest.Build.Ref, "source-ref", "master", "source git ref")
	triggerCmd.PersistentFlags().StringVar(&triggerBuildRequest.Build.DockerfilePath, "dockerfile-path", ".", "Dockerfile path (optional)")
	triggerCmd.PersistentFlags().StringArrayVar(&tags, "tags", []string{}, "image tags (comma-delimited)")
	triggerCmd.PersistentFlags().BoolVar(&triggerBuildRequest.Build.TagWithCommitSha, "tag-sha", true, "additionally tag with git commit SHA (optional)")
	triggerCmd.PersistentFlags().BoolVar(&triggerBuildRequest.SkipIfExists, "skip-if-exists", false, "if build already exists at destination, skip build/push (registry: all tags exist, s3: object exists)")
	triggerCmd.PersistentFlags().StringVar(&cachetype, "build-cache-type", "disabled", "Build cache type (one of: disabled, s3, inline)")
	triggerCmd.PersistentFlags().BoolVar(&cachemaxmode, "build-cache-max-mode", false, "Build cache max mode (see BuildKit docs)")
	triggerCmd.PersistentFlags().StringSliceVar(&buildArgs, "build-arg", []string{}, "Build arg to use for build request")
	triggerCmd.PersistentFlags().StringArrayVar(&imagerepos, "image-repos", []string{}, "Image repositories (comma-separated)")
	triggerCmd.PersistentFlags().BoolVar(&monitorBuild, "monitor", true, "Monitor build after triggering")
	triggerCmd.PersistentFlags().DurationVar(&rpctimeout, "timeout", 30*time.Second, "Timeout for RPC calls")
	triggerCmd.PersistentFlags().DurationVar(&buildtimeout, "build-timeout", 30*time.Minute, "Timeout for build duration")
	triggerCmd.PersistentFlags().StringVar(&triggerBuildRequest.Build.Resources.CpuRequest, "cpu-request", "", "override BuildKit container CPU request (optional)")
	triggerCmd.PersistentFlags().StringVar(&triggerBuildRequest.Build.Resources.MemoryRequest, "mem-request", "", "override BuildKit container memory request (optional)")
	triggerCmd.PersistentFlags().StringVar(&triggerBuildRequest.Build.Resources.CpuLimit, "cpu-limit", "", "override BuildKit container CPU limit (optional)")
	triggerCmd.PersistentFlags().StringVar(&triggerBuildRequest.Build.Resources.MemoryLimit, "mem-limit", "", "override BuildKit container memory limit (optional)")
	RootCmd.AddCommand(triggerCmd)
}

func trigger(cmd *cobra.Command, args []string) error {
	rb, err := client.New(clientops)
	if err != nil {
		return fmt.Errorf("error creating rpc client: %w", err)
	}
	defer rb.Close()

	triggerBuildRequest.Push.Registries = make([]*furanrpc.PushRegistryDefinition, len(imagerepos))
	for i := range imagerepos {
		triggerBuildRequest.Push.Registries[i] = &furanrpc.PushRegistryDefinition{
			Repo: imagerepos[i],
		}
	}

	triggerBuildRequest.Build.Tags = tags

	if !triggerBuildRequest.Build.TagWithCommitSha && len(tags) == 0 {
		return fmt.Errorf("at least one tag is required if not tagging with commit SHA")
	}

	triggerBuildRequest.Build.CacheOptions = &furanrpc.BuildCacheOpts{MaxMode: cachemaxmode}
	switch cachetype {
	case "disabled":
		triggerBuildRequest.Build.CacheOptions.Type = furanrpc.BuildCacheOpts_DISABLED
	case "s3":
		triggerBuildRequest.Build.CacheOptions.Type = furanrpc.BuildCacheOpts_S3
	case "inline":
		triggerBuildRequest.Build.CacheOptions.Type = furanrpc.BuildCacheOpts_INLINE
	default:
		return fmt.Errorf("unknown or invalid build cache type: %v", cachetype)
	}

	ctx, cf := context.WithTimeout(context.Background(), rpctimeout)
	defer cf()

	id, err := rb.StartBuild(ctx, triggerBuildRequest)
	if err != nil {
		return fmt.Errorf("error triggering build: %w", err)
	}

	fmt.Fprintf(os.Stderr, "build started: %v\n", id)

	if monitorBuild {
		ctx2, cf2 := context.WithTimeout(context.Background(), buildtimeout)
		defer cf2()
		mc, err := rb.MonitorBuild(ctx2, id)
		if err != nil {
			return fmt.Errorf("error monitoring build: %w", err)
		}
		var i uint
		for {
			be, err := mc.Recv()
			if err != nil {
				if err == io.EOF {
					break

				}
				return fmt.Errorf("error getting build message: %w", err)
			}
			var status string
			if be.CurrentState != furanrpc.BuildState_RUNNING {
				status = fmt.Sprintf(" (status: %v)", be.CurrentState)
			}
			fmt.Fprintf(os.Stderr, "%v%v\n", be.Message, status)
			i++
		}
		bs, err := rb.GetBuildStatus(context.Background(), id)
		if err != nil {
			return fmt.Errorf("error getting final build status: %w", err)
		}
		duration := " "
		if bs.Started != nil && bs.Completed != nil {
			d := models.TimeFromRPCTimestamp(*bs.Completed).Sub(models.TimeFromRPCTimestamp(*bs.Started))
			duration = fmt.Sprintf(" (duration: %v) ", d)
		}
		fmt.Fprintf(os.Stderr, "build completed: state: %v%v(%d msgs received)\n", bs.State, duration, i)
		return nil
	}

	return nil
}
