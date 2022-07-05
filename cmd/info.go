/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/kubefirst/nebulous/pkg/flare"
	"github.com/spf13/cobra"
	"log"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Provides general host and cli information",
	Long:  `Shows a summary of host details and cli information`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Printf("Kubefirst-cli version: v%s", kubefirstCliVersion)
		log.Printf("OS type: %s", localOs)
		log.Printf("Architecture: %s", localArchitecture)
		log.Printf("$HOME folder: %s", homeFolder)
		log.Printf("Kubectl path: %s", kubectlClientPath)
		log.Printf("Terraform path: %s", terraformPath)
		log.Printf("Kubeconfig path: %s", kubeconfigPath)

		flare.CheckFlareFile(homeFolder)
		flare.CheckKubefirstDir(homeFolder)
		flare.CheckEnvironment()
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// infoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// infoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
