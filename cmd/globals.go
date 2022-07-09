package cmd

import (
	"fmt"
	"log"
	"os"
	"runtime"
)

//Common used strings by all commands
var home, kubectlClientPath, kubeconfigPath, localOs, localArchitecture, terraformPath, helmClientPath string
var dryrunMode bool

//Should this be loaded from somewhere?
var installerEmail = "kubefirst-bot@kubefirst.com"

//setGlobals for all common used properties
func setGlobals() {
	tmphome, err := os.UserHomeDir()
	home = tmphome
	if err != nil {
		log.Panicf("Error Defining home - %s", err)
	}
	localOs = runtime.GOOS
	localArchitecture = runtime.GOARCH
	kubectlClientPath = fmt.Sprintf("%s/.kubefirst/tools/kubectl", home)
	kubeconfigPath = fmt.Sprintf("%s/.kubefirst/gitops/terraform/base/kubeconfig_kubefirst", home)
	terraformPath = fmt.Sprintf("%s/.kubefirst/tools/terraform", home)
	helmClientPath = fmt.Sprintf("%s/.kubefirst/tools/helm", home)
	dryrunMode = false
}
