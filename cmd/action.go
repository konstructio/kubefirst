package cmd

import (
	"fmt"

	"github.com/kubefirst/kubefirst/internal/argocd"
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
		//config := aws.GetConfig("kubefirst-tech")
		clientset, err := k8s.GetClientSet(false, "/Users/scott/.kube/config")
		if err != nil {
			fmt.Println(err)
		}

		err = argocd.ApplyArgoCDKustomize(clientset)
		if err != nil {
			fmt.Println(err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(actionCmd)
}
