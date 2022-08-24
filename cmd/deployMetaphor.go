/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/metaphor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// deployMetaphorCmd represents the deployMetaphor command
var deployMetaphorCmd = &cobra.Command{
	Use:   "deploy-metaphor",
	Short: "Add metaphor applications to the cluster",
	Long:  `TBD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("deployMetaphor called")
		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}

		if viper.GetBool("github.enabled") {
			return metaphor.DeployMetaphorGithub(globalFlags)
		} else {
			return metaphor.DeployMetaphorGitlab(globalFlags)
		}

	},
}

func init() {
	actionCmd.AddCommand(deployMetaphorCmd)
	flagset.DefineGlobalFlags(deployMetaphorCmd)
}
