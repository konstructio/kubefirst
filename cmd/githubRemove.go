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
		gitWrapper := githubWrapper.New()
		err = gitWrapper.RemoveRepo(owner, "gitops")
		if err != nil {
			return err
		}
		err = gitWrapper.RemoveRepo(owner, "metaphor")
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	actionCmd.AddCommand(githubRemoveCmd)

	githubRemoveCmd.Flags().String("github-owner", "", "Github Owner of repos")
	viper.BindPFlag("github.owner", githubRemoveCmd.Flags().Lookup("github.owner"))
	githubRemoveCmd.MarkFlagRequired("github.owner")

	githubRemoveCmd.Flags().String("github-org", "", "Github Org of repos")
	viper.BindPFlag("github.org", githubRemoveCmd.Flags().Lookup("github-org"))
}
