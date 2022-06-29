/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"log"
	"github.com/spf13/cobra"
	"gitlab.kubefirst.io/kubefirst/flare/pkg/flare"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Provide a general overview of host machine and cli",
	Long: `Command used to allow a deeper inspection of the host machine 
	and cli version runnig and its current state. Tool recommended for troubleshooting 
	installations`,
	Run: func(cmd *cobra.Command, args []string) {		
		log.Printf("flare-cli golang utility version: v%s", NebolousVersion)
		log.Printf("OS type: %s", localOs)
		log.Printf("Arch: %s", localArchitecture)
		log.Printf("$HOME folder: %s", home)
		log.Printf("kubectl used: %s", kubectlClientPath)
		log.Printf("terraform used: %s", terraformPath)
		log.Printf("Kubeconfig in use: %s", kubeconfigPath)
		flare.CheckFlareFile(home)
		flare.CheckKubefirstDir(home)	
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
