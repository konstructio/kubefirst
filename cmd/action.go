package cmd

import (
	argocd "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
)

// actionCmd represents the action command
var actionCmd = &cobra.Command{
	Use:   "action",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		argocd.NewFromConfig()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(actionCmd)
}
