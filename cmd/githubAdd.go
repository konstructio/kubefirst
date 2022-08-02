/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// githubAddCmd represents the setupGithub command
var githubAddCmd = &cobra.Command{
	Use:   "add-github",
	Short: "Setup github for kubefirst install",
	Long:  `TBD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("setupGithub called")
		org, err := cmd.Flags().GetString("github-org")
		if err != nil {
			return err
		}
		fmt.Println("Org used:", org)

		gitWrapper := githubWrapper.New()
		gitWrapper.CreatePrivateRepo(org, "gitops", "Kubefirst Gitops")
		gitWrapper.CreatePrivateRepo(org, "metaphor", "Sample Kubefirst App")
		return nil
	},
}

func init() {
	actionCmd.AddCommand(githubAddCmd)

	githubAddCmd.Flags().String("github-org", "", "Github Org of repos")
	viper.BindPFlag("github.org", githubAddCmd.Flags().Lookup("github-org"))

}
