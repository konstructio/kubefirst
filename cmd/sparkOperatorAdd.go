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

// sparkOperatorAddCmd represents the sparkOperator command
var sparkOperatorAddCmd = &cobra.Command{
	Use:   "spark-operator-add",
	Short: "Add spark Capabilities to Cluster",
	Long:  `TBD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("spark-operator-add  called")
		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			log.Print(err)
			return err
		}

		helm.InstallSparkOperator(dryRun)
		k8s.AddPermissionsForSparkOperator("default")
		fmt.Println("Permissions Added")
		return nil
	},
}

func init() {
	addonsCmd.AddCommand(sparkOperatorAddCmd)
	sparkOperatorAddCmd.Flags().Bool("dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
}
