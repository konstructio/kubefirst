/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// githubPopulateCmd represents the githubPopulate command
var githubPopulateCmd = &cobra.Command{
	Use:   "github-populate",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("githubPopulate called")
		config := configs.ReadConfig()
		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}
		log.Println("dry-run:", globalFlags.DryRun)

		if viper.GetBool("github.repo.populated") {
			log.Println("github.repo.populated already executed, skiped")
			return nil
		}
		if globalFlags.DryRun {
			log.Printf("[#99] Dry-run mode, githubPopulateCmd skipped.")
			return nil
		}

		owner := viper.GetString("github.owner")

		fmt.Println("githubPopulate: gitops")
		gitClient.PopulateRepoWithToken(owner, "gitops", fmt.Sprintf("%s/%s", config.K1FolderPath, "gitops"), viper.GetString("github.host"))
		viper.Set("github.gitops-pushed", true)

		fmt.Println("githubPopulate: metaphor")
		gitClient.PopulateRepoWithToken(owner, "metaphor", fmt.Sprintf("%s/%s", config.K1FolderPath, "metaphor"), viper.GetString("github.host"))
		viper.Set("github.metaphor-pushed", true)

		viper.Set("github.repo.populated", true)
		viper.WriteConfig()
		return nil
	},
}

func init() {
	actionCmd.AddCommand(githubPopulateCmd)
	currentCommand := githubPopulateCmd
	flagset.DefineGlobalFlags(currentCommand)

}
