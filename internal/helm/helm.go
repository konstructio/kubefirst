package helm

import (
	"fmt"
	"github.com/kubefirst/nebulous/configs"
	"github.com/spf13/viper"
	"log"
	"os"
	"os/exec"
)

func InstallArgocd(dryRun bool) {
	config := configs.ReadConfig()
	if !viper.GetBool("create.argocd.helm") {
		if dryRun {
			log.Printf("[#99] Dry-run mode, helmInstallArgocd skipped.")
			return
		}
		// ! commenting out until a clean execution is necessary // create namespace
		helmRepoAddArgocd := exec.Command(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "repo", "add", "argo", "https://argoproj.github.io/argo-helm")
		helmRepoAddArgocd.Stdout = os.Stdout
		helmRepoAddArgocd.Stderr = os.Stderr
		err := helmRepoAddArgocd.Run()
		if err != nil {
			log.Panicf("error: could not run helm repo add %s", err)
		}

		helmRepoUpdate := exec.Command(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "repo", "update")
		helmRepoUpdate.Stdout = os.Stdout
		helmRepoUpdate.Stderr = os.Stderr
		err = helmRepoUpdate.Run()
		if err != nil {
			log.Panicf("error: could not helm repo update %s", err)
		}

		helmInstallArgocdCmd := exec.Command(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "upgrade", "--install", "argocd", "--namespace", "argocd", "--create-namespace", "--wait", "--values", fmt.Sprintf("%s/.kubefirst/argocd-init-values.yaml", config.HomePath), "argo/argo-cd")
		helmInstallArgocdCmd.Stdout = os.Stdout
		helmInstallArgocdCmd.Stderr = os.Stderr
		err = helmInstallArgocdCmd.Run()
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
