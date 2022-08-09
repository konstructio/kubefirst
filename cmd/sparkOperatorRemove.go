/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"log"

	"github.com/kubefirst/kubefirst/internal/helm"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/spf13/cobra"
)

// sparkOperatorRemoveCmd represents the removeSparkOperator command
var sparkOperatorRemoveCmd = &cobra.Command{
	Use:   "spark-operator-remove",
	Short: "Remove addons spark operator",
	Long:  `TBD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("remove-spark-operator called")
		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			log.Print(err)
			return err
		}
		k8s.RemovePermissionsForSparkOperator("default")
		helm.UninstallSparkOperator(dryRun)
		return nil
	},
}

func init() {
	addonsCmd.AddCommand(sparkOperatorRemoveCmd)
	sparkOperatorRemoveCmd.Flags().Bool("dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")

}
