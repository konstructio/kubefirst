package helm

import (
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	"log"
)

func InstallArgocd(dryRun bool) {
	config := configs.ReadConfig()
	if !viper.GetBool("create.argocd.helm") {
		if dryRun {
			log.Printf("[#99] Dry-run mode, helmInstallArgocd skipped.")
			return
		}
		// ! commenting out until a clean execution is necessary // create namespace
		_, _, err := pkg.ExecShellReturnStrings(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "repo", "add", "argo", "https://argoproj.github.io/argo-helm")
		if err != nil {
			log.Panicf("error: could not run helm repo add %s", err)
		}

		_, _, err = pkg.ExecShellReturnStrings(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "repo", "update")
		if err != nil {
			log.Panicf("error: could not helm repo update %s", err)
		}

		_, _, err = pkg.ExecShellReturnStrings(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "upgrade", "--install", "argocd", "--namespace", "argocd", "--create-namespace", "--wait", "--values", fmt.Sprintf("%s/argocd-init-values.yaml", config.K1FolderPath), "argo/argo-cd")
		if err != nil {
			log.Panicf("error: could not helm install argocd command %s", err)
		}

		viper.Set("create.argocd.helm", true)
		err = viper.WriteConfig()
		if err != nil {
			log.Panicf("error: could not write to viper config")
		}
	}
}
