/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/spf13/cobra"
)

// destroyAwsGitlabCmd represents the destroyAwsGitlab command
var destroyAwsGitlabCmd = &cobra.Command{
	Use:   "destroy-aws-gitlab",
	Short: "A brief description of your command",
	Long:  `TDB`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("destroy-aws-gitlab called")
		config := configs.ReadConfig()

		destroyFlags, err := flagset.ProcessDestroyFlags(cmd)
		if err != nil {
			log.Println(err)
			return err
		}

		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			log.Println(err)
			return err
		}

		if globalFlags.SilentMode {
			informUser(
				"Silent mode enabled, most of the UI prints wont be showed. Please check the logs for more details.\n",
				globalFlags.SilentMode,
			)
		}
		log.Println(destroyFlags, config)
		return nil
	},
}

func init() {
	clusterCmd.AddCommand(destroyAwsGitlabCmd)
	currentCommand := destroyAwsGitlabCmd
	flagset.DefineGlobalFlags(currentCommand)
	flagset.DefineDestroyFlags(currentCommand)
}
