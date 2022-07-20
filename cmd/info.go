package cmd

import (
	"bytes"
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/spf13/cobra"
	"log"
	"runtime"
	"strings"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "provides general Kubefirst setup data",
	Long:  `Provides machine data, files and folders paths`,
	Run: func(cmd *cobra.Command, args []string) {

		config := configs.ReadConfig()

		var infoSummary bytes.Buffer

		infoSummary.WriteString(strings.Repeat("-", 70))
		infoSummary.WriteString("\nInfo summary:\n")
		infoSummary.WriteString(strings.Repeat("-", 70))

		//infoSummary.WriteString(fmt.Sprintf("Kubefirst CLI version: v%s \n", config.KubefirstVersion))
		infoSummary.WriteString(fmt.Sprintf("\n\nOperational System: %s\n", config.LocalOs))
		infoSummary.WriteString(fmt.Sprintf("Architecture: %s\n", config.LocalArchitecture))
		infoSummary.WriteString(fmt.Sprintf("Go Lang version: v%s \n", runtime.Version()))
		infoSummary.WriteString(fmt.Sprintf("Kubefirst config file: %s\n", config.KubefirstConfigFilePath))
		infoSummary.WriteString(fmt.Sprintf("Kubefirst config folder: %s\n", config.K1FolderPath))
		infoSummary.WriteString(fmt.Sprintf("Kubectl path: %s\n", config.KubectlClientPath))
		infoSummary.WriteString(fmt.Sprintf("Terraform path: %s\n", config.TerraformPath))
		infoSummary.WriteString(fmt.Sprintf("Kubeconfig path: %s\n", config.KubeConfigPath))

		err := configs.CheckKubefirstConfigFile(config)
		if err != nil {
			log.Panic(err)
		}
		err = configs.CheckKubefirstDir(config)
		if err != nil {
			log.Panic(err)
		}

		fmt.Println(reports.StyleMessage(infoSummary.String()))
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
