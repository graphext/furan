package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/dollarshaveclub/furan/pkg/client"
	"github.com/dollarshaveclub/furan/pkg/generated/furanrpc"
	"github.com/dollarshaveclub/furan/pkg/models"
)

// triggerCmd represents the trigger command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List builds",
	Long:  `List active or previous builds according to criteria. Multiple criteria are combined with an implicit AND.`,
	RunE:  list,
}

var listOps furanrpc.ListBuildsRequest

type statusmap map[string]models.BuildStatus

var cliStatuses = statusmap{
	"notstarted":      models.BuildStatusNotStarted,
	"running":         models.BuildStatusRunning,
	"skipped":         models.BuildStatusSkipped,
	"failure":         models.BuildStatusFailure,
	"success":         models.BuildStatusSuccess,
	"cancelrequested": models.BuildStatusCancelRequested,
	"cancelled":       models.BuildStatusCancelled,
}

func (sm statusmap) keys() []string {
	out := make([]string, len(sm))
	i := 0
	for k := range sm {
		out[i] = k
		i++
	}
	return out
}

var smkeys = strings.Join(cliStatuses.keys(), ", ")

var statusstr string
var completedafter, startedafter, completedbefore, startedbefore time.Duration

func init() {
	setclientflags(listCmd)
	listCmd.PersistentFlags().StringVar(&listOps.WithGithubRepo, "with-github-repo", "", "List builds from this GitHub repo")
	listCmd.PersistentFlags().StringVar(&listOps.WithGithubRef, "with-github-ref", "", "List builds from this GitHub ref")
	listCmd.PersistentFlags().StringVar(&listOps.WithImageRepo, "with-image-repo", "", "List builds pushing to this image repo")
	listCmd.PersistentFlags().StringVar(&statusstr, "with-status", "", fmt.Sprintf("List builds with this status (one of: %v)", smkeys))
	listCmd.PersistentFlags().DurationVar(&completedafter, "with-completed-after", 0, "List builds completed after this duration prior to now")
	listCmd.PersistentFlags().DurationVar(&startedafter, "with-started-after", 0, "List builds started after this duration prior to now")
	listCmd.PersistentFlags().DurationVar(&completedbefore, "with-completed-before", 0, "List builds completed before this duration prior to now")
	listCmd.PersistentFlags().DurationVar(&startedbefore, "with-started-before", 0, "List builds started before this duration prior to now")
	RootCmd.AddCommand(listCmd)
}

func timeListOps() {
	now := time.Now().UTC()
	if completedafter != 0 {
		ts := models.RPCTimestampFromTime(now.Add(-1 * completedafter))
		listOps.CompletedAfter = &ts
	}
	if startedafter != 0 {
		ts := models.RPCTimestampFromTime(now.Add(-1 * startedafter))
		listOps.StartedAfter = &ts
	}
	if completedbefore != 0 {
		ts := models.RPCTimestampFromTime(now.Add(-1 * completedbefore))
		listOps.CompletedBefore = &ts
	}
	if startedbefore != 0 {
		ts := models.RPCTimestampFromTime(now.Add(-1 * startedbefore))
		listOps.StartedBefore = &ts
	}
}

func list(cmd *cobra.Command, args []string) error {
	if statusstr != "" {
		s, ok := cliStatuses[statusstr]
		if !ok {
			return fmt.Errorf("invalid or unknown status: %v (wanted one of: %v)", statusstr, smkeys)
		}
		listOps.WithBuildState = s.State()
	}

	timeListOps()

	rb, err := client.New(clientops)
	if err != nil {
		return fmt.Errorf("error creating rpc client: %w", err)
	}
	defer rb.Close()

	ctx := context.Background()

	builds, err := rb.ListBuilds(ctx, listOps)
	if err != nil {
		return err
	}

	// Format in tab-separated columns with a tab stop of 8.
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Fprintln(w, "BUILD ID\tGITHUB REPO\tSTATUS\tSTARTED\tCOMPLETED")
	for _, b := range builds {
		var bid, ghrepo, status, started, completed string
		bid = b.BuildId
		ghrepo = b.BuildRequest.Build.GithubRepo
		status = b.State.String()
		started = models.TimeFromRPCTimestamp(*b.Started).Format(time.RFC3339)
		if b.Completed != nil {
			completed = models.TimeFromRPCTimestamp(*b.Completed).Format(time.RFC3339)
		}
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\n", bid, ghrepo, status, started, completed)
	}
	w.Flush()

	return nil
}
