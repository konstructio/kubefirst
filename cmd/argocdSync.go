package cmd

import (
	"log"

	"github.com/kubefirst/kubefirst/internal/argocd"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// argocdSyncCmd represents the argocdSync command
var argocdSyncCmd = &cobra.Command{
	Use:   "argocdSync",
	Short: "request ArgoCD to synchronize applications",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		// dryRun, err := cmd.Flags().GetBool("dry-run")
		// if err != nil {
		// 	log.Panic(err)
		// }

		dryRun := false

		log.Println("dry run enabled:", dryRun)

		applicationName, _ := cmd.Flags().GetString("app-name")
		refreshToken, _ := cmd.Flags().GetBool("refresh-token")

		authToken := viper.GetString("argocd.admin.apitoken")

		if !refreshToken && authToken == "" {
			log.Panic("uh oh - no argocd auth token found in config, try again with `--refresh-token` ")
		} else {
			log.Println("getting a new argocd session token")
			authToken = argocd.GetArgocdAuthToken(dryRun)
		}
		log.Printf("syncing the %s application", applicationName)
		argocd.SyncArgocdApplication(dryRun, applicationName, authToken)
	},
}

func init() {
	rootCmd.AddCommand(argocdSyncCmd)
	argocdSyncCmd.Flags().String("app-name", "", "gets a new argocd session token")
	argocdSyncCmd.MarkFlagRequired("app-name")
	argocdSyncCmd.Flags().Bool("refresh-token", false, "gets a new argocd session token")
}
