/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/kubefirst/nebulous/pkg/flare"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Provide a general overview of host machine and cli",
	Long: `Command used to allow a deeper inspection of the host machine 
	and cli version runnig and its current state. Tool recommended for troubleshooting 
	installations`,
	Run: func(cmd *cobra.Command, args []string) {		
		fmt.Printf("flare-cli golang utility version: v%s \n", NebolousVersion)
		fmt.Printf("OS type: %s\n", localOs)
		fmt.Printf("Arch: %s\n", localArchitecture)
		fmt.Printf("$HOME folder: %s\n", home)
		fmt.Printf("kubectl used: %s\n", kubectlClientPath)
		fmt.Printf("terraform used: %s\n", terraformPath)
		fmt.Printf("Kubeconfig in use: %s\n", kubeconfigPath)
		flare.CheckFlareFile(home,true)
		flare.CheckKubefirstDir(home,true)	
		flare.CheckEnvironment(true)	
		fmt.Printf("----------- \n")
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
