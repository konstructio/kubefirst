/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"runtime"

	"github.com/kubefirst/kubefirst-api/pkg/configs"
	"github.com/kubefirst/kubefirst/internal/progress"
	"github.com/spf13/cobra"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "provides general Kubefirst setup data",
	Long:  `Provides machine data, files and folders paths`,
	Run: func(cmd *cobra.Command, args []string) {
		config := configs.ReadConfig()

		content := `
##
# Info summary

| Name        						| Value                           	|
| ---         						| ---                             	|
| Operational System   		|  ` + config.LocalOs + `						|
| Architecture 						|  ` + config.LocalArchitecture + `|
| Go Lang version					|  ` + runtime.Version() + `|
| Kubefirst config file		|  ` + config.KubefirstConfigFilePath + `|
| Kubefirst config folder	|  ` + config.K1FolderPath + `|
| Kubefirst Version       |  ` + configs.K1Version + `|
`

		// infoSummary.WriteString(fmt.Sprintf("Kubectl path: %s", config.KubectlClientPath))
		// infoSummary.WriteString(fmt.Sprintf("Terraform path: %s", config.TerraformClientPath))
		// infoSummary.WriteString(fmt.Sprintf("Kubeconfig path: %s", config.KubeConfigPath))

		// if configs.K1Version == "" {
		// 	infoSummary.WriteString("\n\nWarning: It seems you are running kubefirst in development mode,")
		// 	infoSummary.WriteString("  please use LDFLAGS to ensure you use the proper template version and avoid unexpected behavior")
		// }

		progress.Success(content)
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
