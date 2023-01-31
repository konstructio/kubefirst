package cmd

import (
	"fmt"

	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

		clusterName := "mgmt-green"
		k1DirPath := viper.GetString("kubefirst.k1-directory-path")
		kubeconfigPath := viper.GetString("kubefirst.kubeconfig-path")
		kubectlClientPath := viper.GetString("kubefirst.kubectl-client-path")
		registryYamlPath := fmt.Sprintf("%s/gitops/registry/%s/registry-%s.yaml", k1DirPath, clusterName, clusterName)

		err := argocd.KubectlCreateApplication(false, kubeconfigPath, kubectlClientPath, k1DirPath, registryYamlPath)
		if err != nil {
			log.Info().Msgf("Error applying %s application to argocd", registryYamlPath)
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(actionCmd)
}
