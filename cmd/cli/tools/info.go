package tools

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
	"strings"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/spf13/cobra"
)

func InfoCommand() *cobra.Command {
	infoCmd := &cobra.Command{
		Use:   "info",
		Short: "provides general Kubefirst setup data",
		Long:  `Provides machine data, files and folders paths`,
		Run:   RunInfo,
	}

	return infoCmd
}

func RunInfo(cmd *cobra.Command, args []string) {

	config := configs.ReadConfig()

	var infoSummary bytes.Buffer

	infoSummary.WriteString(strings.Repeat("-", 70))
	infoSummary.WriteString("\nInfo summary:\n")
	infoSummary.WriteString(strings.Repeat("-", 70))

	infoSummary.WriteString(fmt.Sprintf("\n\nOperational System: %s\n", config.LocalOs))
	infoSummary.WriteString(fmt.Sprintf("Architecture: %s\n", config.LocalArchitecture))
	infoSummary.WriteString(fmt.Sprintf("Go Lang version: v%s \n", runtime.Version()))
	infoSummary.WriteString(fmt.Sprintf("Kubefirst config file: %s\n", config.KubefirstConfigFilePath))
	infoSummary.WriteString(fmt.Sprintf("Kubefirst config folder: %s\n", config.K1FolderPath))
	infoSummary.WriteString(fmt.Sprintf("Kubectl path: %s\n", config.KubectlClientPath))
	infoSummary.WriteString(fmt.Sprintf("Terraform path: %s\n", config.TerraformClientPath))
	infoSummary.WriteString(fmt.Sprintf("Kubeconfig path: %s\n", config.KubeConfigPath))

	infoSummary.WriteString(fmt.Sprintf("Kubefirst Version: %s\n", configs.K1Version))
	if configs.K1Version == "" {
		infoSummary.WriteString("\n\nWarning: It seems you are running kubefirst in development mode,")
		infoSummary.WriteString("  please use LDFLAGS to ensure you use the proper template version and avoid unexpected behavior")
	}

	err := configs.CheckKubefirstConfigFile(config)
	if err != nil {
		log.Println("Config file check:", err)
	}
	err = configs.CheckKubefirstDir(config)
	if err != nil {
		log.Println("Installer dir check:", err)
	}
	fmt.Printf("----------- \n")

	fmt.Println(reports.StyleMessage(infoSummary.String()))
}

//func initialization() {
//	cmd.rootCmd.AddCommand(infoCmd)
//}