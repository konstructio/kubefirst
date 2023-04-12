/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/runtime/configs"
	"github.com/kubefirst/runtime/pkg/reports"
	"github.com/spf13/cobra"
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

		infoSummary.WriteString(fmt.Sprintf("\n\nOperational System: %s\n", config.LocalOs))
		infoSummary.WriteString(fmt.Sprintf("Architecture: %s\n", config.LocalArchitecture))
		infoSummary.WriteString(fmt.Sprintf("Go Lang version: %s \n", runtime.Version()))
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
			log.Error().Err(err).Msg("config file check")
		}
		err = configs.CheckKubefirstDir(config)
		if err != nil {
			log.Error().Err(err).Msg("installer dir check")
		}
		fmt.Printf("----------- \n")

		fmt.Println(reports.StyleMessage(infoSummary.String()))
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
