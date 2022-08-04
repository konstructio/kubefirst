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

// githubRemoveCmd represents the githubRemove command
var githubRemoveCmd = &cobra.Command{
	Use:   "remove-github",
	Short: "Undo github setup",
	Long:  `TBD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("githubRemove called")
		owner, err := cmd.Flags().GetString("github-owner")
		if err != nil {
			return err
		}
		fmt.Println("Owner used:", owner)
		gitWrapper := githubWrapper.New()
		err = gitWrapper.RemoveRepo(owner, "gitops")
		if err != nil {
			return err
		}
		err = gitWrapper.RemoveRepo(owner, "metaphor")
		if err != nil {
			return err
		}

		viper.Set("github.repo.added", false)
		viper.Set("github.repo.populated", false)
		viper.WriteConfig()
		return nil
	},
}

func init() {
	actionCmd.AddCommand(githubRemoveCmd)

	currentCommand := githubRemoveCmd
	currentCommand.Flags().Bool("dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
	currentCommand.Flags().String("github-owner", "", "Github Owner of repos")
	viper.BindPFlag("github.owner", currentCommand.Flags().Lookup("github.owner"))
	currentCommand.MarkFlagRequired("github.owner")

	currentCommand.Flags().String("github-org", "", "Github Org of repos")
	viper.BindPFlag("github.org", currentCommand.Flags().Lookup("github-org"))
}
