/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

// oneshotCmd represents the oneshot command
var oneshotCmd = &cobra.Command{
	Use:   "oneshot",
	Short: "A oneshot call installer",
	Long: `TBD - A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("oneshot called")

		//test chaining calls in a single place
		versionCmd.Run(cmd, args)
        infoCmd.Run(cmd,args)
	},
}

func init() {
	rootCmd.AddCommand(oneshotCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// oneshotCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// oneshotCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
