/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/spf13/cobra"
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

		sourceFolder := fmt.Sprintf("%s/sample", config.K1FolderPath)
		gitClient.PopulateRepoWithToken(owner, "gitops", sourceFolder, githubHost)
		gitClient.PopulateRepoWithToken(owner, "metaphor", sourceFolder, githubHost)

		return nil
	},
}

func init() {
	actionCmd.AddCommand(githubPopulateCmd)
	githubPopulateCmd.Flags().String("github-owner", "", "Github Owner of repos")
	githubPopulateCmd.Flags().String("github-host", "github.com", "Github repo, usally github.com, but it can change on enterprise customers.")
}
