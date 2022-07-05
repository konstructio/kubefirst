package cmd

import (
	"fmt"
	"os"
	"runtime"
)

//Common used strings by all commands
var homeFolder, kubectlClientPath, kubeconfigPath, localOs, localArchitecture, terraformPath, helmClientPath string
var dryrunMode bool

//Should this be loaded from somewhere?
var installerEmail = "kubefirst-bot@kubefirst.com"

//setGlobals for all common used properties
func setGlobals() {
	tmphome, err := os.UserHomeDir()
	homeFolder = tmphome
	if (err != nil) {
		fmt.Printf("Error Defining homeFolder - %s", err)
		os.Exit(1)
	}
	localOs = runtime.GOOS
	localArchitecture = runtime.GOARCH
	kubectlClientPath = fmt.Sprintf("%s/.kubefirst/tools/kubectl", homeFolder)
	kubeconfigPath = fmt.Sprintf("%s/.kubefirst/gitops/terraform/base/kubeconfig_kubefirst", homeFolder)
	terraformPath = fmt.Sprintf("%s/.kubefirst/tools/terraform", homeFolder)
	helmClientPath = fmt.Sprintf("%s/.kubefirst/tools/helm", homeFolder)
	dryrunMode = false
}