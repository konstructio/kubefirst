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
		flags, err := processGithubAddCmdFlags(cmd)
		if err != nil {
			return err
		}
		fmt.Println("Owner used:", flags.GithubOwner)
		gitWrapper := githubWrapper.New()
		err = gitWrapper.RemoveRepo(flags.GithubOwner, "gitops")
		if err != nil {
			return err
		}
		err = gitWrapper.RemoveRepo(flags.GithubOwner, "metaphor")
		if err != nil {
			return err
		}
		err = gitWrapper.RemoveSSHKey(viper.GetInt64("github.ssh.keyId"))
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
	defineGithubCmdFlags(currentCommand)
	defineGlobalFlags(currentCommand)
	currentCommand.MarkFlagRequired("github-owner")
}
