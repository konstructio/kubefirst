/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"

	"github.com/kubefirst/kubefirst/internal/ciTools"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// addCiCmd represents the addCi command
var addCiCmd = &cobra.Command{
	Use:   "add-ci",
	Short: "Create CI infrastrucute resources in the cloud",
	Long:  `This command must be run to create cloud infrastructure resources in the cloud that allow a CI pipeline to be created and run through Argo Workflows.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("addCi called")

		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}

		ciFlags, err := flagset.ProcessCIFlags(cmd)
		if err != nil {
			return err
		}

		bucketName, err := ciTools.CreateBucket()
		if err != nil {
			return err
		}

		err = ciTools.ApplyTemplates(globalFlags)
		if err != nil {
			return err
		}

		if viper.GetString("gitprovider") == "gitlab" {
			ciTools.DeployOnGitlab(globalFlags, bucketName)
		}

		ciTools.ApplyCITerraform(globalFlags.DryRun, bucketName)

		log.Println(ciFlags)
		return nil
	},
}

func init() {
	actionCmd.AddCommand(addCiCmd)
	flagset.DefineCIFlags(addCiCmd)
	flagset.DefineGlobalFlags(addCiCmd)
}
