package helm

import (
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

// InstallArgocd - install argoCd in a cluster
// it has a retry embeded logic to mitigate network issues when trying to install argoCD
func InstallArgocd(dryRun bool) error {
	config := configs.ReadConfig()
	message := "error installing argo-cd: unexpected state"

	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, helmInstallArgocd skipped.")
		return nil
	}

	if viper.GetBool("argocd.helm.install.complete") {
		log.Info().Msg("[#99] Already created before, helmInstallArgocd skipped.")
		return nil
	}

	// ! commenting out until a clean execution is necessary // create namespace
	// Refers to: https://github.com/kubefirst/kubefirst/issues/434
	totalAttempts := 5
	for i := 0; i < totalAttempts; i++ {
		log.Info().Msgf("Installing Argo-CD, attempt (%d of %d)", i+1, totalAttempts)
		_, _, err := pkg.ExecShellReturnStrings(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "repo", "add", "argo", "https://argoproj.github.io/argo-helm")
		if err != nil {
			log.Error().Err(err).Msg("error: could not run helm repo add")
			message = "error installing argo-cd: add repo"
			continue
		}

		_, _, err = pkg.ExecShellReturnStrings(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "repo", "update")
		if err != nil {
			log.Error().Err(err).Msg("error: could not helm repo update")
			message = "error installing argo-cd: update repo"
			continue
		}

		_, _, err = pkg.ExecShellReturnStrings(config.HelmClientPath, "--kubeconfig", config.KubeConfigPath, "upgrade", "--install", "argocd", "--namespace", "argocd", "--create-namespace", "--version", config.ArgoCDChartHelmVersion, "--wait", "--values", fmt.Sprintf("%s/argocd-init-values.yaml", config.K1FolderPath), "argo/argo-cd")
		if err != nil {
			log.Error().Err(err).Msg("error: could not helm install argocd command")
			message = "error installing argo-cd: install argo-cd"
			continue
		}

		viper.Set("argocd.helm.install.complete", true)
		err = viper.WriteConfig()
		if err != nil {
			log.Error().Err(err).Msg("error: could not write to viper config")
			message = "error installing argo-cd: update config"
			continue
		}
		return nil
	}

	// previous for loop will attempt to install argo, if the attempts fail, it will reach this point, and returns
	// the default error message
	return fmt.Errorf(message)
}

type HelmRepo struct {
	RepoName     string
	ChartName    string
	RepoURL      string
	Namespace    string
	ChartVersion string
}

func AddRepoAndUpdateRepo(dryRun bool, helmClientPath string, helmRepo HelmRepo, kubeconfigPath string) error {
	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, helm.AddRepoAndUpdateRepo skipped.")
		return nil
	}

	log.Info().Msgf("executing `helm repo add %s %s` ", helmRepo.RepoName, helmRepo.RepoURL)
	_, _, err := pkg.ExecShellReturnStrings(helmClientPath, "--kubeconfig", kubeconfigPath, "repo", "add", helmRepo.RepoName, helmRepo.RepoURL)
	if err != nil {
		log.Error().Err(err).Msgf("error adding helm repo %s", helmRepo.RepoName)
		return err
	}

	log.Info().Msg("executing `helm repo update`")
	_, _, err = pkg.ExecShellReturnStrings(helmClientPath, "--kubeconfig", kubeconfigPath, "repo", "update")
	if err != nil {
		log.Error().Err(err).Msgf("error updating helm repo %s", helmRepo.RepoName)
		return err
	}

	return nil
}

// func Install(argoCDInitValuesYamlPath string, dryRun bool, helmClientPath string, helmRepo HelmRepo, kubeconfigPath string) error {
func Install(dryRun bool, helmClientPath string, helmRepo HelmRepo, kubeconfigPath string) error {
	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, helm.Install skipped.")
		return nil
	}

	log.Info().Msgf("executing `helm install %s` and waiting for completion ", helmRepo.ChartName)
	// todo remove `"--set", "fullnameOverride=argocd", "--set", "nameOverride=argocd"` see type ConfigRepo
	//! , "--values", argoCDInitValuesYamlPath,
	a, b, err := pkg.ExecShellReturnStrings(helmClientPath, "--kubeconfig", kubeconfigPath, "upgrade", "--install", helmRepo.ChartName, "--namespace", helmRepo.Namespace, "--create-namespace", "--version", helmRepo.ChartVersion, "--wait", "--set", "fullnameOverride=argocd", "--set", "nameOverride=argocd", fmt.Sprintf("%s/%s", helmRepo.RepoName, helmRepo.ChartName))
	log.Info().Msg(a)
	log.Info().Msg(b)
	if err != nil {
		log.Error().Err(err).Msgf("error: could not helm install %s - %s", helmRepo.ChartName, err.Error())
		return err
	}

	return nil
}
