/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addonsCmd = &cobra.Command{
	Use:   "addons",
	Short: "Addons support",
	Long:  `Support a set of extra resources to be added in an existing cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("addons called")
	},
}

func init() {
	rootCmd.AddCommand(addonsCmd)
}
