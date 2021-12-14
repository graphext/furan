package cmd

import (
	"context"
	"fmt"

	"github.com/dollarshaveclub/furan/v2/pkg/client"
	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
)

// eventsCmd represents the events command
var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Get build events",
	Long:  `Get build events and status information for a current or historical build job from a remote Furan server`,
	RunE:  events,
}

var ids string

func init() {
	setclientflags(eventsCmd)
	eventsCmd.PersistentFlags().StringVar(&ids, "id", "", "build ID (uuid)")
	RootCmd.AddCommand(eventsCmd)
}

func events(cmd *cobra.Command, args []string) error {
	if ids == "" {
		return fmt.Errorf("build id is required")
	}

	bid, err := uuid.FromString(ids)
	if err != nil {
		return fmt.Errorf("invalid build id: %w", err)
	}

	rb, err := client.New(clientops)
	if err != nil {
		return fmt.Errorf("error creating rpc client: %w", err)
	}
	defer rb.Close()

	ctx := context.Background()

	resp, err := rb.GetBuildEvents(ctx, bid)
	if err != nil {
		return fmt.Errorf("error getting build events: %w", err)
	}

	for _, e := range resp.Messages {
		fmt.Println(e)
	}

	fmt.Printf("STATUS: %v\n", resp.CurrentState.String())

	return nil
}
