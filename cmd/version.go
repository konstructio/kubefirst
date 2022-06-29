/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"log"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number for flare",
	Long:  `All software has versions. This is flare's`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Printf("flare-cli golang utility version: v%s", NebolousVersion)
		
	},
}
