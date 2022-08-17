/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"log"

	"github.com/kubefirst/kubefirst/internal/flagset"
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
		fmt.Println("Owner used:", viper.GetString("github.owner"))
		gitWrapper := githubWrapper.New()
		err := gitWrapper.RemoveRepo(viper.GetString("github.owner"), "gitops")
		if err != nil {
			return err
		}
		err = gitWrapper.RemoveRepo(viper.GetString("github.owner"), "metaphor")
		if err != nil {
			return err
		}
		err = gitWrapper.RemoveSSHKey(viper.GetInt64("github.ssh.keyId"))
		if err != nil {
			log.Println("Trying to remove key failed:", err)
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
	flagset.DefineGlobalFlags(currentCommand)
}
