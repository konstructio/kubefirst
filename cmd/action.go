package cmd

import (
	argo "github.com/argoproj/argo-cd/pkg/client/clientset/versioned"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/k8s"
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
		config := aws.GetConfig("kubefirst-tech")

		config, err := k8s.GetClientConfig(false, config.Kubeconfig)
		if err != nil {
			return err
		}

		argo.NewForConfig()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(actionCmd)
}
