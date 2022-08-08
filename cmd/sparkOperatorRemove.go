/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

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
		k8s.RemovePermissionsForSparkOperator("default")
		return nil
	},
}

func init() {
	addonsCmd.AddCommand(sparkOperatorRemoveCmd)

}
