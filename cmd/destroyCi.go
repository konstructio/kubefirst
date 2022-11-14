/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/ciTools"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/spf13/cobra"
)

// destroyCiCmd represents the destroyCi command
var destroyCiCmd = &cobra.Command{
	Use:   "destroy-ci",
	Short: "Destroy CI infrastrucute resources in the cloud",
	Long:  `This command must be executed to destroy infrastructure resources previously created in the cloud to be used by a CI pipeline using Argo Workflows.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("destroyCi called")

		config := configs.ReadConfig()
		ciDirectory := fmt.Sprintf("%s/ci", config.K1FolderPath)

		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}

		ciFlags, err := flagset.ProcessCIFlags(cmd)
		if err != nil {
			return err
		}

		ciTools.DestroyCITerraform(globalFlags.DryRun)

		err = ciTools.DeleteTemplates(globalFlags)
		if err != nil {
			return err
		}

		if ciFlags.DestroyBucket {
			err = ciTools.DestroyBucket()
			if err != nil {
				return err
			}
		}

		err = ciTools.DestroyGitRepository(globalFlags)
		if err != nil {
			log.Panicf("error to destroy ci repostory:  %s", err)
			//return err
		}

		err = os.RemoveAll(ciDirectory)
		if err != nil {
			return fmt.Errorf("unable to delete %q folder, error is: %s", ciDirectory, err)
		}

		log.Println(ciFlags)
		return nil

	},
}

func init() {
	actionCmd.AddCommand(destroyCiCmd)
	flagset.DefineCIFlags(destroyCiCmd)
	flagset.DefineGlobalFlags(destroyCiCmd)
}
