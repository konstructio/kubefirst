package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// argocdSyncCmd represents the argocdSync command
var argocdSyncCmd = &cobra.Command{
	Use:   "argocdSync",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		applicationName, _ := cmd.Flags().GetString("app-name")
		refreshToken, _ := cmd.Flags().GetBool("refresh-token")

		authToken := viper.GetString("argocd.admin.apitoken")

		if !refreshToken && authToken == "" {
			log.Panic("uh oh - no argocd auth token found in config, try again with `--refresh-token` ")
		} else {
			log.Println("getting a new argocd session token")
			authToken = getArgocdAuthToken()
		}
		log.Printf("syncing the %s application", applicationName)
		syncArgocdApplication(applicationName, authToken)
	},
}

func init() {
	rootCmd.AddCommand(argocdSyncCmd)
	argocdSyncCmd.Flags().String("app-name", "", "gets a new argocd session token")
	argocdSyncCmd.MarkFlagRequired("app-name")
	argocdSyncCmd.Flags().Bool("refresh-token", false, "gets a new argocd session token")
}
