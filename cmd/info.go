package cmd

import (
	"fmt"
	"github.com/kubefirst/nebulous/configs"
	"github.com/spf13/cobra"
	"log"
	"runtime"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "provides general Kubefirst setup data",
	Long:  `Provides machine data, files and folders paths`,
	Run: func(cmd *cobra.Command, args []string) {

		config := configs.ReadConfig()

		fmt.Printf("Kubefirst CLI version: v%s \n", config.KubefirstVersion)
		fmt.Printf("Operational System: %s\n", config.LocalOs)
		fmt.Printf("Architecture: %s\n", config.LocalArchitecture)
		fmt.Printf("Go Lang version: v%s \n", runtime.Version())
		fmt.Printf("$HOME folder: %s\n", config.HomePath)
		fmt.Printf("Kubectl path: %s\n", config.KubectlClientPath)
		fmt.Printf("Terraform path: %s\n", config.TerraformPath)
		fmt.Printf("Kubeconfig path: %s\n", config.KubeConfigPath)

		err := configs.CheckFlareFile(config.HomePath)
		if err != nil {
			log.Panic(err)
		}
		err = configs.CheckKubefirstDir(config.HomePath)
		if err != nil {
			log.Panic(err)
		}
		err = configs.CheckEnvironment()
		if err != nil {
			log.Panic(err)
		}
		fmt.Printf("----------- \n")
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
