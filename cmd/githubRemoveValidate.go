/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/spf13/cobra"
)

// githubRemoveValidateCmd represents the githubRemoveValidate command
var githubRemoveValidateCmd = &cobra.Command{
	Use:   "githubRemoveValidate",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("githubRemoveValidate called")

		owner, err := cmd.Flags().GetString("github-owner")
		if err != nil {
			return err
		}
		gitWrapper := githubWrapper.New()
		repoGitops, err := gitWrapper.GetRepo(owner, "gitops")
		//TODO: Improve logic
		if err == nil {
			fmt.Println("gitops not found as expected")
		}
		repoMetaphor, err := gitWrapper.GetRepo(owner, "metaphor")
		if err == nil {
			fmt.Println("gitops not found as expected")
		}

		if repoGitops.GetName() == "gitops" {
			fmt.Println("gitops should be not present")
			return fmt.Errorf("error validating repo: %s ", repoGitops.GetName())
		}

		if repoMetaphor.GetName() == "metaphor" {
			fmt.Println("metaphor should be not present")
			return fmt.Errorf("error validating repo: %s ", repoGitops.GetName())
		}
		return nil
	},
}

func init() {
	actionCmd.AddCommand(githubRemoveValidateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// githubRemoveValidateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// githubRemoveValidateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
