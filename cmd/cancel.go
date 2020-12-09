package cmd

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"

	"github.com/dollarshaveclub/furan/pkg/client"
)

// triggerCmd represents the trigger command
var cancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel a running build",
	Long:  `Cancel a running build in a remote Furan cluster`,
	RunE:  cancel,
}

var cancelBuildID string

func init() {
	setclientflags(cancelCmd)
	cancelCmd.PersistentFlags().StringVar(&cancelBuildID, "build-id", "", "Build ID")
	RootCmd.AddCommand(cancelCmd)
}

func cancel(cmd *cobra.Command, args []string) error {
	rb, err := client.New(clientops)
	if err != nil {
		return fmt.Errorf("error creating rpc client: %w", err)
	}
	defer rb.Close()

	id, err := uuid.FromString(cancelBuildID)
	if err != nil {
		return fmt.Errorf("malformed or invalid build id: %w", err)
	}

	ctx := context.Background()

	err = rb.CancelBuild(ctx, id)
	if err != nil {
		return fmt.Errorf("error sending cancel request: %w", err)
	}

	fmt.Println("build cancellation requested: " + id.String())
	return nil
}
