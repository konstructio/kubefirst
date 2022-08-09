/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"log"

	"github.com/kubefirst/kubefirst/configs"
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
		owner, err := cmd.Flags().GetString("github-owner")
		if err != nil {
			return err
		}

		githubHost, err := cmd.Flags().GetString("github-host")
		if err != nil {
			return err
		}
		dryrun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			return err
		}
		log.Println("dry-run:", dryrun)

		if viper.GetBool("github.repo.populated") {
			log.Println("github.repo.populated already executed, skiped")
			return nil
		}
		if dryrun {
			log.Printf("[#99] Dry-run mode, githubPopulateCmd skipped.")
			return nil
		}

		//sourceFolder := fmt.Sprintf("%s/sample", config.K1FolderPath)
		sourceFolder, err := gitClient.CloneRepoAndDetokenize(config.GitopsTemplateURL, "gitops", "main")
		if err != nil {
			return err
		}
		fmt.Println("githubPopulate:", sourceFolder)
		gitClient.PopulateRepoWithToken(owner, "gitops", sourceFolder, githubHost)
		sourceFolder, err = gitClient.CloneRepoAndDetokenize(config.MetaphorTemplateURL, "metaphor", "main")
		if err != nil {
			return err
		}
		fmt.Println("githubPopulate:", sourceFolder)
		gitClient.PopulateRepoWithToken(owner, "metaphor", sourceFolder, githubHost)

		viper.Set("github.repo.populated", true)
		viper.WriteConfig()
		return nil
	},
}

func init() {
	actionCmd.AddCommand(githubPopulateCmd)
	currentCommand := githubPopulateCmd
	currentCommand.Flags().Bool("dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
	currentCommand.Flags().String("github-owner", "", "Github Owner of repos")
	currentCommand.Flags().String("github-host", "github.com", "Github repo, usally github.com, but it can change on enterprise customers.")
}
