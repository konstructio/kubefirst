package tool

import "github.com/spf13/cobra"

var (
	lastTunnel string
	owner      string
	repo       string
)

func NewCommand() *cobra.Command {

	toolCmd := &cobra.Command{
		Use:   "tool",
		Short: "general tools set",
		Long:  "TBD",
		RunE:  runTool,
		// PostRunE: runPostCivo,
	}

	// wire up new commands
	toolCmd.AddCommand(WebhookUpdater())

	return toolCmd
}

func WebhookUpdater() *cobra.Command {

	webhookUpdater := &cobra.Command{
		Use:     "tool",
		Short:   "comand to be used check/update ngrok based webhooks",
		Long:    "TBD",
		RunE:    runWebhookUpdater,
		PreRunE: validateWebhookUpdater,
		// PostRunE: runPostCivo,
	}

	webhookUpdater.Flags().StringVar(&owner, "repo", "", "repository that will observed fro changes on tunnels")
	webhookUpdater.Flags().StringVar(&repo, "owner", "", "owner of repository that will observed fro changes on tunnels, organization or user")

	return webhookUpdater
}
