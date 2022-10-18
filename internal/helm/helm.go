package helm

import (
	"fmt"
	"log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

// InstallArgocd - install argoCd in a cluster
// it has a retry embeded logic to mitigate network issues when trying to install argoCD
func InstallArgocd(dryRun bool) error {
	config := configs.ReadConfig()
	message := "error installing argo-cd: unexpected state"
	if !viper.GetBool("argocd.helm.install.complete") {
		if dryRun {
			log.Printf("[#99] Dry-run mode, helmInstallArgocd skipped.")
			return nil
		}
		// ! commenting out until a clean execution is necessary // create namespace
		// Refers to: https://github.com/kubefirst/kubefirst/issues/434
		totalAttempts := 5
		for i := 0; i < totalAttempts; i++ {
			log.Printf("Installing Argo-CD, attempt (%d of %d)", i+1, totalAttempts)
			_, _, err := pkg.ExecShellReturnStrings(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "repo", "add", "argo", "https://argoproj.github.io/argo-helm")
			if err != nil {
				log.Printf("error: could not run helm repo add %s", err)
				message = "error installing argo-cd: add repo"
				continue
			}

			_, _, err = pkg.ExecShellReturnStrings(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "repo", "update")
			if err != nil {
				log.Printf("error: could not helm repo update %s", err)
				message = "error installing argo-cd: update repo"
				continue
			}

			_, _, err = pkg.ExecShellReturnStrings(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "upgrade", "--install", "argocd", "--namespace", "argocd", "--create-namespace", "--version", config.ArgoCDChartHelmVersion, "--wait", "--values", fmt.Sprintf("%s/argocd-init-values.yaml", config.K1FolderPath), "argo/argo-cd")
			if err != nil {
				log.Printf("error: could not helm install argocd command %s", err)
				message = "error installing argo-cd: install argo-cd"
				continue
			}

			viper.Set("argocd.helm.install.complete", true)
			err = viper.WriteConfig()
			if err != nil {
				log.Printf("error: could not write to viper config")
				message = "error installing argo-cd: update config"
				continue
			}
		}
	} else {
		log.Printf("[#99] Already created before, helmInstallArgocd skipped.")
		return nil
	}
	return fmt.Errorf(message)
}

type HelmRepo struct {
	RepoName     string
	ChartName    string
	RepoURL      string
	Namespace    string
	ChartVersion string
}

func AddRepoAndUpdateRepo(dryRun bool, helmRepo HelmRepo) error {
	if dryRun {
		log.Printf("[#99] Dry-run mode, helm.AddRepoAndUpdateRepo skipped.")
		return nil
	}

	config := configs.ReadConfig()

	log.Printf("executing `helm repo add %s %s` ", helmRepo.RepoName, helmRepo.RepoURL)
	_, _, err := pkg.ExecShellReturnStrings(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "repo", "add", helmRepo.RepoName, helmRepo.RepoURL)
	if err != nil {
		log.Printf("error adding helm repo %s", helmRepo.RepoName)
		return err
	}
	viper.Set("argocd.helm.repo.added", true)
	viper.WriteConfig()

	log.Println("executing `helm repo update`")
	_, _, err = pkg.ExecShellReturnStrings(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "repo", "update")
	if err != nil {
		log.Printf("error updating helm repo %s", helmRepo.RepoName)
		return err
	}
	viper.Set("argocd.helm.repo.updated", true)
	viper.WriteConfig()
	return nil
}

func Install(dryRun bool, helmRepo HelmRepo) error {
	if dryRun {
		log.Printf("[#99] Dry-run mode, helm.Install skipped.")
		return nil
	}

	config := configs.ReadConfig()

	log.Printf("executing `helm install %s` and waiting for completion ", helmRepo.ChartName)
	// todo remove `"--set", "fullnameOverride=argocd", "--set", "nameOverride=argocd"` see type ConfigRepo
	_, _, err := pkg.ExecShellReturnStrings(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "upgrade", "--install", helmRepo.ChartName, "--namespace", helmRepo.Namespace, "--create-namespace", "--version", helmRepo.ChartVersion, "--wait", "--set", "fullnameOverride=argocd", "--set", "nameOverride=argocd", "--values", config.ArgoCDInitValuesYamlPath, fmt.Sprintf("%s/%s", helmRepo.RepoName, helmRepo.ChartName))
	if err != nil {
		log.Printf("error: could not helm install %s - %s", helmRepo.ChartName, err)
	}
	viper.Set("argocd.helm.install.complete", true)
	viper.WriteConfig()
	return nil
}
