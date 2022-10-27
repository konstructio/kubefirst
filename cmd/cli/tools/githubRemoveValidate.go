/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package tools

import (
	"fmt"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/spf13/cobra"
)

// githubRemoveValidateCmd represents the githubRemoveValidate command
var githubRemoveValidateCmd = &cobra.Command{
	Use:   "remove-github-validate",
	Short: "TBD",
	Long:  `TBD`,
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
		repoMetaphorGo, err := gitWrapper.GetRepo(owner, "metaphor-go")
		if err == nil {
			fmt.Println("gitops not found as expected")
		}
		repoMetaphorFrontend, err := gitWrapper.GetRepo(owner, "metaphor-frontend")
		if err == nil {
			fmt.Println("gitops not found as expected")
		}

		if repoGitops.GetName() == "gitops" {
			fmt.Println("gitops should be not present")
			return fmt.Errorf("error validating repo: %s ", repoGitops.GetName())
		}

		if repoMetaphor.GetName() == "metaphor" {
			fmt.Println("metaphor should be not present")
			return fmt.Errorf("error validating repo: %s ", repoMetaphor.GetName())
		}

		if repoMetaphorGo.GetName() == "metaphor-go" {
			fmt.Println("metaphor should be not present")
			return fmt.Errorf("error validating repo: %s ", repoMetaphorGo.GetName())
		}

		if repoMetaphorFrontend.GetName() == "metaphor-frontend" {
			fmt.Println("metaphor should be not present")
			return fmt.Errorf("error validating repo: %s ", repoMetaphorFrontend.GetName())
		}
		return nil
	},
}

func init() {
	//cmd.actionCmd.AddCommand(githubRemoveValidateCmd)
}
