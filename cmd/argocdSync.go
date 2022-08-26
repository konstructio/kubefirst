package cmd

import (
	"errors"
	"log"

	"github.com/kubefirst/kubefirst/internal/argocd"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// argocdSyncCmd request ArgoCD to synchronize applications
var argocdSyncCmd = &cobra.Command{
	Use:   "argocdSync",
	Short: "request ArgoCD to synchronize applications",
	Long:  `During installation we must wait Argo to be ready, this command get a token and try to sync Argo application`,

	RunE: func(cmd *cobra.Command, args []string) error {

		applicationName, err := cmd.Flags().GetString("app-name")
		if err != nil {
			return err
		}

		refreshToken, err := cmd.Flags().GetBool("refresh-token")
		if err != nil {
			return err
		}

		authToken := viper.GetString("argocd.admin.apitoken")
		if !refreshToken && authToken == "" {
			return errors.New("uh oh - no argocd auth token found in config, try again with --refresh-token")
		}

		log.Println("getting a new argocd session token")
		authToken = argocd.GetArgocdAuthToken(false)

		log.Printf("syncing the %s application", applicationName)
		argocd.SyncArgocdApplication(false, applicationName, authToken)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(argocdSyncCmd)
	argocdSyncCmd.Flags().String("app-name", "", "gets a new argocd session token")
	err := argocdSyncCmd.MarkFlagRequired("app-name")
	if err != nil {
		log.Println(err)
		return
	}
	argocdSyncCmd.Flags().Bool("refresh-token", false, "gets a new argocd session token")
}
