/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kubefirst/kubefirst/internal/helm"
	k3dint "github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/downloadManager"
	"github.com/kubefirst/runtime/pkg/helpers"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/kubefirst/runtime/pkg/k8s"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	clusterName = "kubefirst-console"
)

const (
	consoleURL        = "https://console.kubefirst.dev"
	helmChartName     = "kubefirst"
	helmChartRepoName = "kubefirst"
	helmChartRepoURL  = "https://charts.kubefirst.com"
	helmChartVersion  = "0.0.28"
	namespace         = "kubefirst"
	secretName        = "kubefirst-initial-secrets"
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
			dbDestination := k3dint.MongoDestinationChooser()
			var dbHost, dbUser, dbPassword string
			switch dbDestination {
			case "atlas":
				fmt.Println("MongoDB Atlas Host String: ")
				fmt.Scanln(&dbHost)

				fmt.Println("MongoDB Atlas Username: ")
				fmt.Scanln(&dbUser)

				fmt.Println("MongoDB Atlas Password: ")
				fmt.Scanln(&dbPassword)

				fmt.Println()
			case "in-cluster":
			default:
				log.Fatalf("%s is not a valid option", dbDestination)
			}

			helpers.DisplayLogHints()

			log.Infof("%s/%s", k3d.LocalhostOS, k3d.LocalhostARCH)

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

			// Download mkcert
			mkcertClient := fmt.Sprintf("%s/mkcert", toolsDir)
			_, err = os.Stat(mkcertClient)
			if err != nil {
				log.Info("Downloading mkcert...")
				mkcertDownloadURL := fmt.Sprintf(
					"https://github.com/FiloSottile/mkcert/releases/download/%s/mkcert-%s-%s-%s",
					"v1.4.4",
					"v1.4.4",
					k3d.LocalhostOS,
					k3d.LocalhostARCH,
				)
				err = downloadManager.DownloadFile(mkcertClient, mkcertDownloadURL)
				if err != nil {
					log.Fatalf("error while trying to download mkcert: %s", err)
				}
				err = os.Chmod(mkcertClient, 0755)
				if err != nil {
					log.Fatal(err.Error())
				}
			} else {
				log.Info("mkcert is already installed, continuing")
			}

			// Create k3d cluster
			kubeconfigPath := fmt.Sprintf("%s/.k1/%s/kubeconfig", homeDir, clusterName)
			_, _, err = pkg.ExecShellReturnStrings(
				k3dClient,
				"cluster",
				"get",
				clusterName,
			)
			if err != nil {
				log.Warn("k3d cluster does not exist and will be created")
				log.Info("Creating k3d cluster for Kubefirst console and API...")
				err = k3d.ClusterCreateConsoleAPI(
					clusterName,
					kubeconfigPath,
					k3dClient,
					fmt.Sprintf("%s/kubeconfig", dir),
				)
				if err != nil {
					msg := fmt.Sprintf("error creating k3d cluster: %s", err)
					log.Fatal(msg)
				}
				log.Info("k3d cluster for Kubefirst console and API created successfully")

				// Wait for traefik
				kcfg := k8s.CreateKubeConfig(false, kubeconfigPath)
				log.Info("Waiting for traefik...")
				traefikDeployment, err := k8s.ReturnDeploymentObject(
					kcfg.Clientset,
					"app.kubernetes.io/name",
					"traefik",
					"kube-system",
					240,
				)
				if err != nil {
					log.Fatalf("error looking for traefik: %s", err)
				}
				_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, traefikDeployment, 120)
				if err != nil {
					log.Fatalf("error waiting for traefik: %s", err)
				}
			}

			// Establish Kubernetes client for console cluster
			kcfg := k8s.CreateKubeConfig(false, kubeconfigPath)

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
				kubeconfigPath,
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
					"kubefirst/kubefirst",
					"--set",
					"console.ingress.createTraefikRoute=true",
					"--set",
					"kubefirst-api.installMethod=kubefirst-launch",
				}

				switch k3d.LocalhostARCH {
				case "arm64":
					installFlags = append(installFlags, "--set")
					installFlags = append(installFlags, "kubefirst-api.image.hook.tag=arm64")
				}

				switch dbDestination {
				case "in-cluster":
					installFlags = append(installFlags, "--create-namespace")
					installFlags = append(installFlags, "--set")
					installFlags = append(installFlags, "mongodb.enabled=true")

					if k3d.LocalhostARCH == "arm64" {
						installFlags = append(installFlags, "--set")
						installFlags = append(installFlags, "mongodb.image.repository=arm64v8/mongo,mongodb.image.tag=latest,mongodb.persistence.mountPath=/data/db,mongodb.extraEnvVarsSecret=kubefirst-initial-secrets")
					}
				case "atlas":
					installFlags = append(installFlags, "--set")
					installFlags = append(installFlags, "mongodb.enabled=false")
					installFlags = append(installFlags, "--set")
					installFlags = append(installFlags, fmt.Sprintf("kubefirst-api.existingSecret=%s", secretName))
					installFlags = append(installFlags, "--set")
					installFlags = append(installFlags, fmt.Sprintf("kubefirst-api.atlasDbHost=%s", dbHost))
					installFlags = append(installFlags, "--set")
					installFlags = append(installFlags, fmt.Sprintf("kubefirst-api.atlasDbUsername=%s", dbUser))

					// Create Namespace
					_, err = kcfg.Clientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
					if err == nil {
						log.Info("kubernetes Namespace already created - skipping")
					} else if strings.Contains(err.Error(), "not found") {
						_, err = kcfg.Clientset.CoreV1().Namespaces().Create(context.Background(), &v1.Namespace{
							ObjectMeta: metav1.ObjectMeta{
								Name: namespace,
							},
						}, metav1.CreateOptions{})
						if err != nil {
							log.Fatalf("error creating kubernetes secret for initial secret: %s", err)
						}
						log.Info("Created Kubernetes Namespace for kubefirst")
					}

					// Create Secret
					_, err = kcfg.Clientset.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
					if err == nil {
						log.Infof("kubernetes secret %s/%s already created - skipping", namespace, secretName)
					} else if strings.Contains(err.Error(), "not found") {
						_, err = kcfg.Clientset.CoreV1().Secrets(namespace).Create(context.Background(), &v1.Secret{
							Type: "Opaque",
							ObjectMeta: metav1.ObjectMeta{
								Name:      secretName,
								Namespace: namespace,
							},
							Data: map[string][]byte{
								"mongodb-root-password": []byte(dbPassword),
							},
						}, metav1.CreateOptions{})
						if err != nil {
							log.Fatalf("error creating kubernetes secret for initial secret: %s", err)
						}
						log.Info("Created Kubernetes Secret for database authentication")
					}
				}

				// Install helm chart
				a, b, err := pkg.ExecShellReturnStrings(helmClient, installFlags...)
				if err != nil {
					log.Fatalf("error installing helm chart: %s %s %s", err, a, b)
				}

				log.Info("Kubefirst console helm chart installed successfully")
			} else {
				log.Info("Kubefirst console helm chart already installed")
			}

			// Wait for API Deployment Pods to transition to Running
			log.Info("Waiting for Kubefirst API Deployment...")
			apiDeployment, err := k8s.ReturnDeploymentObject(
				kcfg.Clientset,
				"app.kubernetes.io/name",
				"kubefirst-api",
				"kubefirst",
				240,
			)
			if err != nil {
				log.Fatalf("error looking for kubefirst api: %s", err)
			}
			_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, apiDeployment, 120)
			if err != nil {
				log.Fatalf("error waiting for kubefirst api: %s", err)
			}

			// Generate certificate for console
			sslPemDir := fmt.Sprintf("%s/ssl", dir)
			if _, err := os.Stat(sslPemDir); os.IsNotExist(err) {
				err := os.MkdirAll(sslPemDir, os.ModePerm)
				if err != nil {
					log.Warnf("%s directory already exists, continuing", sslPemDir)
				}
			}
			log.Info("Certificate directory created")

			mkcertPemDir := fmt.Sprintf("%s/%s/pem", sslPemDir, "kubefirst.dev")
			if _, err := os.Stat(mkcertPemDir); os.IsNotExist(err) {
				err := os.MkdirAll(mkcertPemDir, os.ModePerm)
				if err != nil {
					log.Warnf("%s directory already exists, continuing", mkcertPemDir)
				}
			}

			fullAppAddress := "console.kubefirst.dev"
			certFileName := mkcertPemDir + "/" + "kubefirst-console" + "-cert.pem"
			keyFileName := mkcertPemDir + "/" + "kubefirst-console" + "-key.pem"

			_, _, err = pkg.ExecShellReturnStrings(
				mkcertClient,
				"-cert-file",
				certFileName,
				"-key-file",
				keyFileName,
				"kubefirst.dev",
				fullAppAddress,
			)
			if err != nil {
				log.Fatalf("error generating certificate for console: %s", err)
			}

			//* read certificate files
			certPem, err := os.ReadFile(fmt.Sprintf("%s/%s-cert.pem", mkcertPemDir, "kubefirst-console"))
			if err != nil {
				log.Fatalf("error generating certificate for console: %s", err)
			}
			keyPem, err := os.ReadFile(fmt.Sprintf("%s/%s-key.pem", mkcertPemDir, "kubefirst-console"))
			if err != nil {
				log.Fatalf("error generating certificate for console: %s", err)
			}

			_, err = kcfg.Clientset.CoreV1().Secrets(namespace).Get(context.Background(), "kubefirst-console-tls", metav1.GetOptions{})
			if err == nil {
				log.Infof("kubernetes secret %s/%s already created - skipping", namespace, "kubefirst-console")
			} else if strings.Contains(err.Error(), "not found") {
				_, err = kcfg.Clientset.CoreV1().Secrets(namespace).Create(context.Background(), &v1.Secret{
					Type: "kubernetes.io/tls",
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-tls", "kubefirst-console"),
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"tls.crt": []byte(certPem),
						"tls.key": []byte(keyPem),
					},
				}, metav1.CreateOptions{})
				if err != nil {
					log.Fatalf("error creating kubernetes secret for cert: %s", err)
				}
				log.Info("Created Kubernetes Secret for certificate")
			}

			log.Infof("Kubefirst Console is now available! %s", consoleURL)

			log.Warn("Kubefirst has generated local certificates for use with the console using `mkcert`.")
			log.Warn("If you experience certificate errors when accessing the console, please run the following command: ")
			log.Warnf("	%s -install", mkcertClient)
			log.Warn()
			log.Warn("For more information on `mkcert`, check out: https://github.com/FiloSottile/mkcert")

			err = pkg.OpenBrowser(consoleURL)
			if err != nil {
				log.Errorf("error attempting to open console in browser: %s", err)
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
