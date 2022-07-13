package cmd

import (
	"github.com/kubefirst/kubefirst/configs"
	"github.com/spf13/cobra"
	"log"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print the version number for kubefirst-cli",
	Long:  `All software has versions. This is kubefirst's`,
	Run: func(cmd *cobra.Command, args []string) {
		config := configs.ReadConfig()
		log.Printf("kubefirst-cli golang utility version: v%s", config.KubefirstVersion)
	},
}
