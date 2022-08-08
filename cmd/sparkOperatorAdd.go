/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

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
		k8s.AddPermissionsForSparkOperator("default")
		fmt.Println("Permissions Added")
		return nil
	},
}

func init() {
	addonsCmd.AddCommand(sparkOperatorAddCmd)
}
