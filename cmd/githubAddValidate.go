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

// githubAddValidate represents the githubValidate command
var githubAddValidate = &cobra.Command{
	Use:   "add-github-validate",
	Short: "Validate if github setup was create as expcted",
	Long:  `TBD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("githubValidate called")
		owner, err := cmd.Flags().GetString("github-owner")
		if err != nil {
			return err
		}

		fmt.Println("Owner used:", owner)

		gitWrapper := githubWrapper.New()
		repoGitops, err := gitWrapper.GetRepo(owner, "gitops")
		if err != nil {
			return err
		}
		repoMetaphor, err := gitWrapper.GetRepo(owner, "metaphor")
		if err != nil {
			return err
		}

		if repoGitops.GetName() != "gitops" {
			return fmt.Errorf("error validating repo: %s ", repoGitops.GetName())
		}

		if repoMetaphor.GetName() != "metaphor" {
			return fmt.Errorf("error validating repo: %s ", repoGitops.GetName())
		}
		return nil
	},
}

func init() {
	actionCmd.AddCommand(githubAddValidate)

	githubAddValidate.Flags().String("github-owner", "", "Github Owner of repos")
	viper.BindPFlag("github.owner", githubAddValidate.Flags().Lookup("github.owner"))
	githubAddValidate.MarkFlagRequired("github.owner")

}
