/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package launch

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/konstructio/kubefirst-api/pkg/configs"
	shell "github.com/konstructio/kubefirst-api/pkg/shell"
	pkg "github.com/konstructio/kubefirst-api/pkg/utils"

	"github.com/konstructio/kubefirst-api/pkg/downloadManager"
	"github.com/konstructio/kubefirst-api/pkg/k3d"
	"github.com/konstructio/kubefirst-api/pkg/k8s"
	"github.com/konstructio/kubefirst/internal/helm"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Describes the local kubefirst console cluster name
var consoleClusterName = "kubefirst-console"

// Up
func Up(ctx context.Context, additionalHelmFlags []string, inCluster, useTelemetry bool) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting user's home directory: %w", err)
	}

	dir := fmt.Sprintf("%s/.k1/%s", homeDir, consoleClusterName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			log.Info().Msgf("%q directory already exists, continuing", dir)
		}
	}
	toolsDir := fmt.Sprintf("%s/tools", dir)
	if _, err := os.Stat(toolsDir); os.IsNotExist(err) {
		err := os.MkdirAll(toolsDir, os.ModePerm)
		if err != nil {
			log.Info().Msgf("%q directory already exists, continuing", toolsDir)
		}
	}

	log.Info().Msgf("%s/%s", k3d.LocalhostOS, k3d.LocalhostARCH)
	// Download k3d
	k3dClient := fmt.Sprintf("%s/k3d", toolsDir)
	_, err = os.Stat(k3dClient)
	if err != nil {
		log.Info().Msg("Downloading k3d...")
		k3dDownloadURL := fmt.Sprintf(
			"https://github.com/k3d-io/k3d/releases/download/%s/k3d-%s-%s",
			k3d.K3dVersion,
			k3d.LocalhostOS,
			k3d.LocalhostARCH,
		)
		err = downloadManager.DownloadFile(k3dClient, k3dDownloadURL)
		if err != nil {
			return fmt.Errorf("error while trying to download k3d: %w", err)
		}
		err = os.Chmod(k3dClient, 0o755)
		if err != nil {
			return fmt.Errorf("error changing permissions of k3d client: %w", err)
		}
	} else {
		log.Info().Msg("k3d is already installed, continuing")
	}

	// Download helm
	helmClient := fmt.Sprintf("%s/helm", toolsDir)
	_, err = os.Stat(helmClient)
	if err != nil {
		log.Info().Msg("Downloading helm...")
		helmVersion := "v3.12.0"
		helmDownloadURL := fmt.Sprintf(
			"https://get.helm.sh/helm-%s-%s-%s.tar.gz",
			helmVersion,
			k3d.LocalhostOS,
			k3d.LocalhostARCH,
		)
		helmDownloadTarGzPath := fmt.Sprintf("%s/helm.tar.gz", toolsDir)
		err = downloadManager.DownloadFile(helmDownloadTarGzPath, helmDownloadURL)
		if err != nil {
			return fmt.Errorf("error while trying to download helm: %w", err)
		}
		helmTarDownload, err := os.Open(helmDownloadTarGzPath)
		if err != nil {
			return fmt.Errorf("could not read helm download content: %w", err)
		}
		downloadManager.ExtractFileFromTarGz(
			helmTarDownload,
			fmt.Sprintf("%s-%s/helm", k3d.LocalhostOS, k3d.LocalhostARCH),
			helmClient,
		)
		err = os.Chmod(helmClient, 0o755)
		if err != nil {
			return fmt.Errorf("error changing permissions of helm client: %w", err)
		}
		os.Remove(helmDownloadTarGzPath)
	} else {
		log.Info().Msg("helm is already installed, continuing")
	}

	// Download mkcert
	mkcertClient := fmt.Sprintf("%s/mkcert", toolsDir)
	_, err = os.Stat(mkcertClient)
	if err != nil {
		log.Info().Msg("Downloading mkcert...")
		mkcertDownloadURL := fmt.Sprintf(
			"https://github.com/FiloSottile/mkcert/releases/download/%s/mkcert-%s-%s-%s",
			"v1.4.4",
			"v1.4.4",
			k3d.LocalhostOS,
			k3d.LocalhostARCH,
		)
		err = downloadManager.DownloadFile(mkcertClient, mkcertDownloadURL)
		if err != nil {
			return fmt.Errorf("error while trying to download mkcert: %w", err)
		}
		err = os.Chmod(mkcertClient, 0o755)
		if err != nil {
			return fmt.Errorf("error changing permissions of mkcert client: %w", err)
		}
	} else {
		log.Info().Msg("mkcert is already installed, continuing")
	}

	// Create k3d cluster
	kubeconfigPath := fmt.Sprintf("%s/.k1/%s/kubeconfig", homeDir, consoleClusterName)
	_, _, err = shell.ExecShellReturnStrings(
		k3dClient,
		"cluster",
		"get",
		consoleClusterName,
	)
	if err != nil {
		log.Warn().Msg("k3d cluster does not exist and will be created")
		log.Info().Msg("Creating k3d cluster for Kubefirst console and API...")
		err = k3d.ClusterCreateConsoleAPI(
			consoleClusterName,
			fmt.Sprintf("%s/.k1", homeDir),
			k3dClient,
			fmt.Sprintf("%s/kubeconfig", dir),
		)
		if err != nil {
			return fmt.Errorf("error creating k3d cluster: %w", err)
		}

		log.Info().Msg("k3d cluster for Kubefirst console and API created successfully")

		// Wait for traefik
		kcfg, err := k8s.CreateKubeConfig(false, kubeconfigPath)
		if err != nil {
			return fmt.Errorf("error creating kubernetes client: %w", err)
		}

		log.Info().Msg("Waiting for traefik...")
		traefikDeployment, err := k8s.ReturnDeploymentObject(
			kcfg.Clientset,
			"app.kubernetes.io/name",
			"traefik",
			"kube-system",
			240,
		)
		if err != nil {
			return fmt.Errorf("error looking for traefik: %w", err)
		}
		_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, traefikDeployment, 120)
		if err != nil {
			return fmt.Errorf("error waiting for traefik: %w", err)
		}
	}

	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		_, _, err := shell.ExecShellReturnStrings(
			k3dClient,
			"kubeconfig",
			"get",
			consoleClusterName,
			"--output",
			kubeconfigPath,
		)
		if err != nil {
			return fmt.Errorf("error getting kubeconfig: %w", err)
		}
	}

	// Establish Kubernetes client for console cluster
	kcfg, err := k8s.CreateKubeConfig(false, kubeconfigPath)
	if err != nil {
		return fmt.Errorf("error creating kubernetes client: %w", err)
	}

	// Determine if helm chart repository has already been added
	res, _, err := shell.ExecShellReturnStrings(
		helmClient,
		"repo",
		"list",
		"-o",
		"yaml",
	)
	if err != nil {
		return fmt.Errorf("error listing current helm repositories: %w", err)
	}

	var existingHelmRepositories []helm.Repo
	repoExists := false

	err = yaml.Unmarshal([]byte(res), &existingHelmRepositories)
	if err != nil {
		return fmt.Errorf("could not get existing helm repositories: %w", err)
	}

	for _, repo := range existingHelmRepositories {
		if repo.Name == helmChartRepoName && repo.URL == helmChartRepoURL {
			repoExists = true
		}
	}

	if !repoExists {
		// Add helm chart repository
		_, _, err = shell.ExecShellReturnStrings(
			helmClient,
			"repo",
			"add",
			helmChartRepoName,
			helmChartRepoURL,
		)
		if err != nil {
			return fmt.Errorf("error adding helm chart repository: %w", err)
		}
		log.Info().Msg("Added Kubefirst helm chart repository")
	} else {
		log.Info().Msg("Kubefirst helm chart repository already added")
	}

	// Update helm chart repository locally
	_, _, err = shell.ExecShellReturnStrings(
		helmClient,
		"repo",
		"update",
	)
	if err != nil {
		return fmt.Errorf("error updating helm chart repository: %w", err)
	}
	log.Info().Msg("Kubefirst helm chart repository updated")

	// Determine if helm release has already been installed
	res, _, err = shell.ExecShellReturnStrings(
		helmClient,
		"--kubeconfig",
		kubeconfigPath,
		"list",
		"-o",
		"yaml",
		"-A",
	)
	if err != nil {
		return fmt.Errorf("error listing current helm releases: %w", err)
	}

	var existingHelmReleases []helm.Release
	chartInstalled := false

	err = yaml.Unmarshal([]byte(res), &existingHelmReleases)
	if err != nil {
		return fmt.Errorf("could not get existing helm releases: %w", err)
	}
	for _, release := range existingHelmReleases {
		if release.Name == helmChartName {
			chartInstalled = true
		}
	}

	kubefirstTeam := os.Getenv("KUBEFIRST_TEAM")
	kubefirstTeamInfo := os.Getenv("KUBEFIRST_TEAM_INFO")

	if !chartInstalled {
		installFlags := []string{
			"install",
			"--kubeconfig",
			kubeconfigPath,
			"--namespace",
			namespace,
			helmChartName,
			"--version",
			helmChartVersion,
			"konstruct/kubefirst",
			"--set",
			"console.ingress.createTraefikRoute=true",
			"--set",
			fmt.Sprintf("global.kubefirstVersion=%s", configs.K1Version),
			"--set",
			"global.cloudProvider=k3d",
			"--set",
			"global.clusterType=bootstrap",
			"--set",
			"global.domainName=kubefirst.dev",
			"--set",
			"global.installMethod=kubefirst-launch",
			"--set",
			"global.kubefirstClient=cli",
			"--set",
			fmt.Sprintf("global.kubefirstTeam=%s", kubefirstTeam),
			"--set",
			fmt.Sprintf("global.kubefirstTeamInfo=%s", kubefirstTeamInfo),
			"--set",
			fmt.Sprintf("global.useTelemetry=%s", strconv.FormatBool(useTelemetry)),
			"--set",
			"kubefirst-api.includeVolume=true",
			"--set",
			"kubefirst-api.extraEnv.IN_CLUSTER=true",
			"--set",
			"kubefirst-api-ee.extraEnv.IN_CLUSTER=true",
			"--set",
			"kubefirst-api.serviceAccount.createClusterRoleBinding=true",
			"--set",
			"kubefirst-api-ee.serviceAccount.createClusterRoleBinding=true",
			"--devel",
		}

		if len(additionalHelmFlags) > 0 {
			for _, f := range additionalHelmFlags {
				installFlags = append(installFlags, "--set")
				installFlags = append(installFlags, f)
			}
		}

		installFlags = append(installFlags, "--create-namespace")

		// Install helm chart
		_, _, err := shell.ExecShellReturnStrings(helmClient, installFlags...)
		if err != nil {
			return fmt.Errorf("error installing helm chart: %w", err)
		}

		log.Info().Msg("Kubefirst console helm chart installed successfully")
	} else {
		log.Info().Msg("Kubefirst console helm chart already installed")
	}

	// Wait for API Deployment Pods to transition to Running
	log.Info().Msg("Waiting for Kubefirst API Deployment...")
	apiDeployment, err := k8s.ReturnDeploymentObject(
		kcfg.Clientset,
		"app.kubernetes.io/name",
		"kubefirst-api",
		"kubefirst",
		240,
	)
	if err != nil {
		return fmt.Errorf("error looking for kubefirst api: %w", err)
	}

	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, apiDeployment, 300)
	if err != nil {
		return fmt.Errorf("error waiting for kubefirst api: %w", err)
	}

	// Generate certificate for console
	sslPemDir := fmt.Sprintf("%s/ssl", dir)
	if _, err := os.Stat(sslPemDir); os.IsNotExist(err) {
		err := os.MkdirAll(sslPemDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("error creating directory for certificates: %w", err)
		}
	}
	log.Info().Msg("Certificate directory created")

	mkcertPemDir := fmt.Sprintf("%s/%s/pem", sslPemDir, "kubefirst.dev")
	if _, err := os.Stat(mkcertPemDir); os.IsNotExist(err) {
		err := os.MkdirAll(mkcertPemDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("error creating directory for certificates: %w", err)
		}
	}

	fullAppAddress := "console.kubefirst.dev"
	certFileName := mkcertPemDir + "/" + "kubefirst-console" + "-cert.pem"
	keyFileName := mkcertPemDir + "/" + "kubefirst-console" + "-key.pem"

	_, _, err = shell.ExecShellReturnStrings(
		mkcertClient,
		"-cert-file",
		certFileName,
		"-key-file",
		keyFileName,
		"kubefirst.dev",
		fullAppAddress,
	)
	if err != nil {
		return fmt.Errorf("error generating certificate for console: %w", err)
	}

	// * read certificate files
	certPem, err := os.ReadFile(fmt.Sprintf("%s/%s-cert.pem", mkcertPemDir, "kubefirst-console"))
	if err != nil {
		return fmt.Errorf("error reading certificate for console: %w", err)
	}
	keyPem, err := os.ReadFile(fmt.Sprintf("%s/%s-key.pem", mkcertPemDir, "kubefirst-console"))
	if err != nil {
		return fmt.Errorf("error reading key for console: %w", err)
	}

	_, err = kcfg.Clientset.CoreV1().Secrets(namespace).Get(ctx, "kubefirst-console-tls", metav1.GetOptions{})
	if err == nil {
		log.Info().Msg(fmt.Sprintf("kubernetes secret %q already created - skipping", "kubefirst-console"))
	} else if strings.Contains(err.Error(), "not found") {
		_, err = kcfg.Clientset.CoreV1().Secrets(namespace).Create(ctx, &v1.Secret{
			Type: "kubernetes.io/tls",
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-tls", "kubefirst-console"),
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"tls.crt": certPem,
				"tls.key": keyPem,
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("error creating kubernetes secret for cert: %w", err)
		}
		time.Sleep(5 * time.Second)
		log.Info().Msg("Created Kubernetes Secret for certificate")
	}

	if !inCluster {
		log.Info().Msg(fmt.Sprintf("Kubefirst Console is now available! %q", consoleURL))

		log.Warn().Msg("Kubefirst has generated local certificates for use with the console using `mkcert`.")
		log.Warn().Msg("If you experience certificate errors when accessing the console, please run the following command:")
		log.Warn().Msg(fmt.Sprintf("	%q -install", mkcertClient))
		log.Warn().Msg("For more information on `mkcert`, check out: https://github.com/FiloSottile/mkcert")

		log.Info().Msg("To remove Kubefirst Console and the k3d cluster it runs in, please run the following command:")
		log.Info().Msg("kubefirst launch down")

		err = pkg.OpenBrowser(consoleURL)
		if err != nil {
			log.Printf("error attempting to open console in browser: %v", err)
		}
	}

	viper.Set("launch.deployed", true)
	viper.WriteConfig()

	return nil
}

// Down destroys a k3d cluster for Kubefirst console and API
func Down(_ bool) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting user's home directory: %w", err)
	}

	log.Info().Msg("Deleting k3d cluster for Kubefirst console and API")

	dir := fmt.Sprintf("%s/.k1/%s", homeDir, consoleClusterName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("cluster %q directory does not exist", dir)
	}
	toolsDir := fmt.Sprintf("%s/tools", dir)
	k3dClient := fmt.Sprintf("%s/k3d", toolsDir)

	_, _, err = shell.ExecShellReturnStrings(k3dClient, "cluster", "delete", consoleClusterName)
	if err != nil {
		return fmt.Errorf("error deleting k3d cluster: %w", err)
	}

	log.Info().Msg("k3d cluster for Kubefirst console and API deleted successfully")

	log.Info().Msg(fmt.Sprintf("Deleting cluster directory at %q", dir))
	err = os.RemoveAll(dir)
	if err != nil {
		return fmt.Errorf("error removing directory at %q: %w", dir, err)
	}

	viper.Set("kubefirst", "")
	viper.Set("flags", "")
	viper.Set("launch", "")

	viper.WriteConfig()

	return nil
}
