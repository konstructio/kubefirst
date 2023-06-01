/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/kubefirst/kubefirst/internal/helm"
	k3dint "github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/downloadManager"
	"github.com/kubefirst/runtime/pkg/helpers"
	"github.com/kubefirst/runtime/pkg/k3d"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	clusterName = "kubefirst-console"
)

const (
	helmChartName     = "kubefirst"
	helmChartRepoName = "kubefirst"
	helmChartRepoURL  = "https://charts.kubefirst.com"
	helmChartVersion  = "0.0.12"
)

func LaunchCommand() *cobra.Command {
	launchCommand := &cobra.Command{
		Use:   "launch",
		Short: "create a local k3d cluster and launch the kubefirst console and api in it",
		Long:  "create a local k3d cluster and launch the kubefirst console and api in it",
	}

	// wire up new commands
	launchCommand.AddCommand(launchUp(), launchDown())

	return launchCommand
}

// launchUp creates a new k3d cluster with Kubefirst console and API
func launchUp() *cobra.Command {
	launchUpCmd := &cobra.Command{
		Use:              "up",
		Short:            "launch new console and api instance",
		TraverseChildren: true,
		Run: func(cmd *cobra.Command, args []string) {
			choice := k3dint.MongoDestinationChooser()
			switch choice {
			case "atlas":
			case "in-cluster":
			default:
				log.Fatalf("%s is not a valid option", choice)
			}

			helpers.DisplayLogHints()

			homeDir, err := os.UserHomeDir()
			if err != nil {
				log.Fatalf("something went wrong getting home path: %s", err)
			}
			dir := fmt.Sprintf("%s/.k1/%s", homeDir, clusterName)
			// todo: this should probably delete an existing folder?
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				err := os.MkdirAll(dir, os.ModePerm)
				if err != nil {
					log.Infof("%s directory already exists, continuing", dir)
				}
			}
			toolsDir := fmt.Sprintf("%s/tools", dir)
			if _, err := os.Stat(toolsDir); os.IsNotExist(err) {
				err := os.MkdirAll(toolsDir, os.ModePerm)
				if err != nil {
					log.Infof("%s directory already exists, continuing", toolsDir)
				}
			}

			// Download k3d
			k3dClient := fmt.Sprintf("%s/k3d", toolsDir)
			_, err = os.Stat(k3dClient)
			if err != nil {
				log.Info("Downloading k3d...")
				k3dDownloadUrl := fmt.Sprintf(
					"https://github.com/k3d-io/k3d/releases/download/%s/k3d-%s-%s",
					k3d.K3dVersion,
					k3d.LocalhostOS,
					k3d.LocalhostARCH,
				)
				err = downloadManager.DownloadFile(k3dClient, k3dDownloadUrl)
				if err != nil {
					log.Fatalf("error while trying to download k3d: %s", err)
				}
				err = os.Chmod(k3dClient, 0755)
				if err != nil {
					log.Fatal(err.Error())
				}
			} else {
				log.Info("k3d is already installed, continuing")
			}

			// Download helm
			helmClient := fmt.Sprintf("%s/helm", toolsDir)
			_, err = os.Stat(helmClient)
			if err != nil {
				log.Info("Downloading helm...")
				helmVersion := "v3.12.0"
				helmDownloadUrl := fmt.Sprintf(
					"https://get.helm.sh/helm-%s-%s-%s.tar.gz",
					helmVersion,
					k3d.LocalhostOS,
					k3d.LocalhostARCH,
				)
				helmDownloadTarGzPath := fmt.Sprintf("%s/helm.tar.gz", toolsDir)
				err = downloadManager.DownloadFile(helmDownloadTarGzPath, helmDownloadUrl)
				if err != nil {
					log.Fatalf("error while trying to download helm: %s", err)
				}
				helmTarDownload, err := os.Open(helmDownloadTarGzPath)
				if err != nil {
					log.Fatalf("could not read helm download content")

				}
				downloadManager.ExtractFileFromTarGz(
					helmTarDownload,
					fmt.Sprintf("%s-%s/helm", k3d.LocalhostOS, k3d.LocalhostARCH),
					helmClient,
				)
				err = os.Chmod(helmClient, 0755)
				if err != nil {
					log.Fatal(err.Error())
				}
				os.Remove(helmDownloadTarGzPath)
			} else {
				log.Info("helm is already installed, continuing")
			}

			// Create k3d cluster
			_, _, err = pkg.ExecShellReturnStrings(
				k3dClient,
				"cluster",
				"get",
				clusterName,
			)
			if err != nil {
				log.Warn("k3d cluster does not exist and will be created")
				log.Info("Creating k3d cluster for Kubefirst console and API")
				err = k3d.ClusterCreateConsoleAPI(
					clusterName,
					fmt.Sprintf("%s/.k1/%s/kubeconfig", homeDir, clusterName),
					k3dClient,
					fmt.Sprintf("%s/kubeconfig", dir),
				)
				if err != nil {
					msg := fmt.Sprintf("error creating k3d cluster: %s", err)
					log.Fatal(msg)
				}
				log.Info("k3d cluster for Kubefirst console and API created successfully")
			}

			// Determine if helm chart repository has already been added
			res, _, err := pkg.ExecShellReturnStrings(
				helmClient,
				"repo",
				"list",
				"-o",
				"yaml",
			)
			if err != nil {
				log.Errorf("error listing current helm repositories: %s", err)
			}

			var existingHelmRepositories []helm.HelmRepo
			repoExists := false

			err = yaml.Unmarshal([]byte(res), &existingHelmRepositories)
			if err != nil {
				log.Fatalf("could not get existing helm repositories: %s", err)
			}
			for _, repo := range existingHelmRepositories {
				if repo.Name == helmChartRepoName && repo.URL == helmChartRepoURL {
					repoExists = true
				}
			}

			if !repoExists {
				// Add helm chart repository
				_, _, err = pkg.ExecShellReturnStrings(
					helmClient,
					"repo",
					"add",
					helmChartRepoName,
					helmChartRepoURL,
				)
				if err != nil {
					log.Errorf("error adding helm chart repository: %s", err)
				}
				log.Info("Added Kubefirst helm chart repository")
			} else {
				log.Info("Kubefirst helm chart repository already added")
			}

			// Update helm chart repository locally
			_, _, err = pkg.ExecShellReturnStrings(
				helmClient,
				"repo",
				"update",
			)
			if err != nil {
				log.Errorf("error updating helm chart repository: %s", err)
			}
			log.Info("Kubefirst helm chart repository updated")

			// Determine if helm release has already been installed
			res, _, err = pkg.ExecShellReturnStrings(
				helmClient,
				"--kubeconfig",
				fmt.Sprintf("%s/.k1/%s/kubeconfig", homeDir, clusterName),
				"list",
				"-o",
				"yaml",
				"-A",
			)
			if err != nil {
				log.Errorf("error listing current helm repositories: %s", err)
			}

			var existingHelmReleases []helm.HelmRelease
			chartInstalled := false

			err = yaml.Unmarshal([]byte(res), &existingHelmReleases)
			if err != nil {
				log.Fatalf("could not get existing helm releases: %s", err)
			}
			for _, release := range existingHelmReleases {
				if release.Name == helmChartName {
					chartInstalled = true
				}
			}

			fmt.Println(k3d.LocalhostOS)
			fmt.Println(k3d.LocalhostARCH)

			if !chartInstalled {
				installFlags := []string{
					"install",
					"--kubeconfig",
					fmt.Sprintf("%s/.k1/%s/kubeconfig", homeDir, clusterName),
					"--namespace",
					"kubefirst",
					helmChartName,
					"--create-namespace",
					"--version",
					helmChartVersion,
					"kubefirst/kubefirst",
				}
				if k3d.LocalhostARCH == "arm64" {
					installFlags = append(installFlags, "--set")
					installFlags = append(installFlags, "mongodb.image.repository=arm64v8/mongo,mongodb.image.tag=latest,mongodb.persistence.mountPath=/data/db,mongodb.extraEnvVarsSecret=kubefirst-initial-secrets")
				}

				// Install helm chart
				a, b, err := pkg.ExecShellReturnStrings(helmClient, installFlags...)
				if err != nil {
					log.Errorf("error installing helm chart: %s %s %s", err, a, b)
				}

				log.Info("Kubefirst console helm chart installed successfully")
			} else {
				log.Info("Kubefirst console helm chart already installed")
			}
		},
	}

	return launchUpCmd
}

// launchDown destroys a k3d cluster for Kubefirst console and API
func launchDown() *cobra.Command {
	launchDownCmd := &cobra.Command{
		Use:              "down",
		Short:            "remove console and api instance",
		TraverseChildren: true,
		Run: func(cmd *cobra.Command, args []string) {
			helpers.DisplayLogHints()

			homeDir, err := os.UserHomeDir()
			if err != nil {
				log.Fatalf("something went wrong getting home path: %s", err)
			}

			log.Info("Deleting k3d cluster for Kubefirst console and API")

			dir := fmt.Sprintf("%s/.k1/%s", homeDir, clusterName)
			toolsDir := fmt.Sprintf("%s/tools", dir)
			k3dClient := fmt.Sprintf("%s/k3d", toolsDir)

			_, _, err = pkg.ExecShellReturnStrings(k3dClient, "cluster", "delete", clusterName)
			if err != nil {
				log.Fatal("error deleting k3d cluster")
			}

			log.Info("k3d cluster for Kubefirst console and API deleted successfully")

			log.Infof("Deleting cluster directory at %s", dir)
			err = os.RemoveAll(dir)
			if err != nil {
				log.Warnf("unable to remove directory at %s", dir)
			}
		},
	}

	return launchDownCmd
}
