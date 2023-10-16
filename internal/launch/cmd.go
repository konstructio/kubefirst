/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package launch

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kubefirst/kubefirst/internal/cluster"
	"github.com/kubefirst/kubefirst/internal/helm"
	k3dint "github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/progress"
	"github.com/kubefirst/runtime/configs"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/db"
	"github.com/kubefirst/runtime/pkg/downloadManager"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/term"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// Describes the local kubefirst console cluster name
	consoleClusterName = "kubefirst-console"
	// HTTP client
	httpClient = http.Client{
		Timeout: time.Second * 2,
	}
)

// Up
func Up(additionalHelmFlags []string, inCluster bool, useTelemetry bool) {
	if viper.GetBool("launch.deployed") {
		progress.Error("Kubefirst console has already been deployed. To start over, run `kubefirst launch down` to completely remove the existing console.")
	}

	if !inCluster {
		progress.DisplayLogHints(10)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		progress.Error(fmt.Sprintf("something went wrong getting home path: %s", err))
	}
	dir := fmt.Sprintf("%s/.k1/%s", homeDir, consoleClusterName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			log.Info().Msgf("%s directory already exists, continuing", dir)
		}
	}
	toolsDir := fmt.Sprintf("%s/tools", dir)
	if _, err := os.Stat(toolsDir); os.IsNotExist(err) {
		err := os.MkdirAll(toolsDir, os.ModePerm)
		if err != nil {
			log.Info().Msgf("%s directory already exists, continuing", toolsDir)
		}
	}

	dbInitialized := viper.GetBool("launch.database-initialized")
	var dbHost, dbUser, dbPassword string

	progress.AddStep("Initialize database")

	if !dbInitialized {
		dbDestination := k3dint.MongoDestinationChooser(inCluster)
		switch dbDestination {
		case "atlas":
			fmt.Println("MongoDB Atlas Host String: ")
			fmt.Scanln(&dbHost)

			fmt.Printf("\nMongoDB Atlas Username: ")
			fmt.Scanln(&dbUser)

			fmt.Printf("\nMongoDB Atlas Password: ")
			dbPasswordInput, err := term.ReadPassword(0)
			if err != nil {
				progress.Error(fmt.Sprintf("error parsing password: %s", err))
			}

			dbPassword = string(dbPasswordInput)
			dbHost = strings.Replace(dbHost, "mongodb+srv://", "", -1)

			// Verify database connectivity
			mdbcl := db.Connect(&db.MongoDBClientParameters{
				HostType: dbDestination,
				Host:     dbHost,
				Username: dbUser,
				Password: dbPassword,
			})
			err = mdbcl.TestDatabaseConnection()
			if err != nil {
				progress.Error(fmt.Sprintf("Error validating Mongo credentials: %s", err))
			}
			mdbcl.Client.Disconnect(mdbcl.Context)

			log.Info().Msg("MongoDB Atlas credentials verified")

			viper.Set("launch.database-destination", "atlas")
			viper.Set("launch.database-initialized", true)
			viper.WriteConfig()
		case "in-cluster":
			viper.Set("launch.database-destination", "in-cluster")
			viper.Set("launch.database-initialized", true)
			viper.WriteConfig()
		default:
			progress.Error(fmt.Sprintf("%s is not a valid option", dbDestination))
		}
	} else {
		log.Info().Msg("Database has already been initialized, skipping")
	}

	progress.CompleteStep("Initialize database")

	log.Info().Msgf("%s/%s", k3d.LocalhostOS, k3d.LocalhostARCH)

	progress.AddStep("Download k3d")

	// Download k3d
	k3dClient := fmt.Sprintf("%s/k3d", toolsDir)
	_, err = os.Stat(k3dClient)
	if err != nil {
		log.Info().Msg("Downloading k3d...")
		k3dDownloadUrl := fmt.Sprintf(
			"https://github.com/k3d-io/k3d/releases/download/%s/k3d-%s-%s",
			k3d.K3dVersion,
			k3d.LocalhostOS,
			k3d.LocalhostARCH,
		)
		err = downloadManager.DownloadFile(k3dClient, k3dDownloadUrl)
		if err != nil {
			progress.Error(fmt.Sprintf("error while trying to download k3d: %s", err))
		}
		err = os.Chmod(k3dClient, 0755)
		if err != nil {
			log.Fatal().Msg(err.Error())
		}
	} else {
		log.Info().Msgf("k3d is already installed, continuing")
	}
	progress.CompleteStep("Download k3d")

	// Download helm
	helmClient := fmt.Sprintf("%s/helm", toolsDir)
	_, err = os.Stat(helmClient)
	if err != nil {
		log.Info().Msg("Downloading helm...")
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
			progress.Error(fmt.Sprintf("error while trying to download helm: %s", err))
		}
		helmTarDownload, err := os.Open(helmDownloadTarGzPath)
		if err != nil {
			progress.Error(fmt.Sprintf("could not read helm download content"))

		}
		downloadManager.ExtractFileFromTarGz(
			helmTarDownload,
			fmt.Sprintf("%s-%s/helm", k3d.LocalhostOS, k3d.LocalhostARCH),
			helmClient,
		)
		err = os.Chmod(helmClient, 0755)
		if err != nil {
			log.Fatal().Msg(err.Error())
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
			progress.Error(fmt.Sprintf("error while trying to download mkcert: %s", err))
		}
		err = os.Chmod(mkcertClient, 0755)
		if err != nil {
			progress.Error(err.Error())
		}
	} else {
		log.Info().Msg("mkcert is already installed, continuing")
	}

	progress.AddStep("Create k3d cluster")

	// Create k3d cluster
	kubeconfigPath := fmt.Sprintf("%s/.k1/%s/kubeconfig", homeDir, consoleClusterName)
	_, _, err = pkg.ExecShellReturnStrings(
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
			kubeconfigPath,
			k3dClient,
			fmt.Sprintf("%s/kubeconfig", dir),
		)
		if err != nil {
			msg := fmt.Sprintf("error creating k3d cluster: %s", err)
			progress.Error(msg)
		}
		log.Info().Msg("k3d cluster for Kubefirst console and API created successfully")

		// Wait for traefik
		kcfg := k8s.CreateKubeConfig(false, kubeconfigPath)
		log.Info().Msg("Waiting for traefik...")
		traefikDeployment, err := k8s.ReturnDeploymentObject(
			kcfg.Clientset,
			"app.kubernetes.io/name",
			"traefik",
			"kube-system",
			240,
		)
		if err != nil {
			progress.Error(fmt.Sprintf("error looking for traefik: %s", err))
		}
		_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, traefikDeployment, 120)
		if err != nil {
			progress.Error(fmt.Sprintf("error waiting for traefik: %s", err))
		}
	}

	progress.CompleteStep("Create k3d cluster")

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
		log.Fatal().Msgf("error listing current helm repositories: %s", err)
	}

	var existingHelmRepositories []helm.HelmRepo
	repoExists := false

	err = yaml.Unmarshal([]byte(res), &existingHelmRepositories)
	if err != nil {
		progress.Error(fmt.Sprintf("could not get existing helm repositories: %s", err))
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
			log.Error().Msgf("error adding helm chart repository: %s", err)
		}
		log.Info().Msg("Added Kubefirst helm chart repository")
	} else {
		log.Info().Msg("Kubefirst helm chart repository already added")
	}

	// Update helm chart repository locally
	_, _, err = pkg.ExecShellReturnStrings(
		helmClient,
		"repo",
		"update",
	)
	if err != nil {
		log.Error().Msgf("error updating helm chart repository: %s", err)
	}
	log.Info().Msg("Kubefirst helm chart repository updated")

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
		log.Error().Msgf("error listing current helm repositories: %s", err)
	}

	var existingHelmReleases []helm.HelmRelease
	chartInstalled := false

	err = yaml.Unmarshal([]byte(res), &existingHelmReleases)
	if err != nil {
		progress.Error(fmt.Sprintf("could not get existing helm releases: %s", err))
	}
	for _, release := range existingHelmReleases {
		if release.Name == helmChartName {
			chartInstalled = true
		}
	}

	kubefirstTeam := os.Getenv("KUBEFIRST_TEAM")
	kubefirstTeamInfo := os.Getenv("KUBEFIRST_TEAM_INFO")

	progress.AddStep("Installing Kubefirst")

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
			fmt.Sprintf("global.kubefirstVersion=%s", configs.K1Version),
			"--set",
			"kubefirst-api.installMethod=kubefirst-launch",
			"--set",
			"kubefirst-api.env[0].name=IS_CLUSTER_ZERO",
			"--set",
			"kubefirst-api.env[0].value='true'",
			"--set",
			fmt.Sprintf("kubefirst-api.kubefirstTeam=%s", kubefirstTeam),
			"--set",
			fmt.Sprintf("kubefirst-api.kubefirstTeamInfo=%s", kubefirstTeamInfo),
			"--set",
			fmt.Sprintf("kubefirst-api.useTelemetry=%s", strconv.FormatBool(useTelemetry)),
		}

		if len(additionalHelmFlags) > 0 {
			for _, f := range additionalHelmFlags {
				installFlags = append(installFlags, "--set")
				installFlags = append(installFlags, f)
			}
		}

		switch viper.GetString("launch.database-destination") {
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
				log.Info().Msg("kubernetes Namespace already created - skipping")
			} else if strings.Contains(err.Error(), "not found") {
				_, err = kcfg.Clientset.CoreV1().Namespaces().Create(context.Background(), &v1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: namespace,
					},
				}, metav1.CreateOptions{})
				if err != nil {
					progress.Error(fmt.Sprintf("error creating kubernetes secret for initial secret: %s", err))
				}
				log.Info().Msg("Created Kubernetes Namespace for kubefirst")
			}

			// Create Secret
			_, err = kcfg.Clientset.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
			if err == nil {
				log.Info().Msg(fmt.Sprintf("kubernetes secret %s/%s already created - skipping", namespace, secretName))
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
					progress.Error(fmt.Sprintf("error creating kubernetes secret for initial secret: %s", err))
				}
				log.Info().Msg("Created Kubernetes Secret for database authentication")
			}
		}

		// Install helm chart
		a, b, err := pkg.ExecShellReturnStrings(helmClient, installFlags...)
		if err != nil {
			progress.Error(fmt.Sprintf("error installing helm chart: %s %s %s", err, a, b))
		}

		log.Info().Msg("Kubefirst console helm chart installed successfully")
	} else {
		log.Info().Msg("Kubefirst console helm chart already installed")
	}

	progress.CompleteStep("Installing Kubefirst")

	progress.AddStep("Waiting for kubefirst Deployment")

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
		progress.Error(fmt.Sprintf("error looking for kubefirst api: %s", err))
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, apiDeployment, 300)
	if err != nil {
		progress.Error(fmt.Sprintf("error waiting for kubefirst api: %s", err))
	}

	// Generate certificate for console
	sslPemDir := fmt.Sprintf("%s/ssl", dir)
	if _, err := os.Stat(sslPemDir); os.IsNotExist(err) {
		err := os.MkdirAll(sslPemDir, os.ModePerm)
		if err != nil {
			log.Warn().Msgf("%s directory already exists, continuing", sslPemDir)
		}
	}
	log.Info().Msg("Certificate directory created")

	mkcertPemDir := fmt.Sprintf("%s/%s/pem", sslPemDir, "kubefirst.dev")
	if _, err := os.Stat(mkcertPemDir); os.IsNotExist(err) {
		err := os.MkdirAll(mkcertPemDir, os.ModePerm)
		if err != nil {
			log.Warn().Msgf("%s directory already exists, continuing", mkcertPemDir)
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
		progress.Error(fmt.Sprintf("error generating certificate for console: %s", err))
	}

	//* read certificate files
	certPem, err := os.ReadFile(fmt.Sprintf("%s/%s-cert.pem", mkcertPemDir, "kubefirst-console"))
	if err != nil {
		progress.Error(fmt.Sprintf("error generating certificate for console: %s", err))
	}
	keyPem, err := os.ReadFile(fmt.Sprintf("%s/%s-key.pem", mkcertPemDir, "kubefirst-console"))
	if err != nil {
		progress.Error(fmt.Sprintf("error generating certificate for console: %s", err))
	}

	_, err = kcfg.Clientset.CoreV1().Secrets(namespace).Get(context.Background(), "kubefirst-console-tls", metav1.GetOptions{})
	if err == nil {
		log.Info().Msg(fmt.Sprintf("kubernetes secret %s/%s already created - skipping", namespace, "kubefirst-console"))
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
			progress.Error(fmt.Sprintf("error creating kubernetes secret for cert: %s", err))
		}
		log.Info().Msg("Created Kubernetes Secret for certificate")
	}

	progress.CompleteStep("Waiting for kubefirst Deployment")

	if !inCluster {
		log.Info().Msg(fmt.Sprintf("Kubefirst Console is now available! %s", consoleURL))

		log.Warn().Msgf("Kubefirst has generated local certificates for use with the console using `mkcert`.")
		log.Warn().Msgf("If you experience certificate errors when accessing the console, please run the following command: ")
		log.Warn().Msgf("	%s -install", mkcertClient)
		log.Warn().Msgf("")
		log.Warn().Msgf("For more information on `mkcert`, check out: https://github.com/FiloSottile/mkcert")

		log.Info().Msg("To remove Kubefirst Console and the k3d cluster it runs in, please run the following command: ")
		log.Info().Msg("kubefirst launch down")

		err = pkg.OpenBrowser(consoleURL)
		if err != nil {
			log.Error().Msgf("error attempting to open console in browser: %s", err)
		}
	}

	viper.Set("launch.deployed", true)
	viper.WriteConfig()

	if !inCluster {
		progress.Success(`
###
#### :tada: Success` + "`Kubefirst Cluster is now up and running`")
	}
}

// Down destroys a k3d cluster for Kubefirst console and API
func Down(inCluster bool) {
	if !inCluster {
		progress.DisplayLogHints(1)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		progress.Error(fmt.Sprintf("something went wrong getting home path: %s", err))
	}

	log.Info().Msg("Deleting k3d cluster for Kubefirst console and API")

	dir := fmt.Sprintf("%s/.k1/%s", homeDir, consoleClusterName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		progress.Error(fmt.Sprintf("cluster %s directory does not exist", dir))
	}
	toolsDir := fmt.Sprintf("%s/tools", dir)
	k3dClient := fmt.Sprintf("%s/k3d", toolsDir)

	_, _, err = pkg.ExecShellReturnStrings(k3dClient, "cluster", "delete", consoleClusterName)
	if err != nil {
		progress.Error(fmt.Sprintf("error deleting k3d cluster: %s", err))
	}

	log.Info().Msg("k3d cluster for Kubefirst console and API deleted successfully")

	log.Info().Msg(fmt.Sprintf("Deleting cluster directory at %s", dir))
	err = os.RemoveAll(dir)
	if err != nil {
		log.Warn().Msgf("unable to remove directory at %s", dir)
	}

	viper.Set("kubefirst", "")
	viper.Set("flags", "")
	viper.Set("launch", "")

	viper.WriteConfig()

	if !inCluster {
		successMsg := `
###
#### :tada: Success` + "`Your K3D kubefirst platform has been destroyed.`"
		progress.Success(successMsg)
	}
}

// ListClusters makes a request to the console API to list created clusters
func ListClusters() {
	clusters, err := cluster.GetClusters()

	err = displayFormattedClusterInfo(clusters)
	if err != nil {
		progress.Error(fmt.Sprintf("error printing cluster list: %s", err))
	}
}

// DeleteCluster makes a request to the console API to delete a single cluster
func DeleteCluster(managedClusterName string) {
	err := cluster.DeleteCluster(managedClusterName)

	if err != nil {
		progress.Error(fmt.Sprintf("error: cluster %s not found\n", managedClusterName))
	}

	deleteMessage := `
##
### Submitted request to delete cluster` + fmt.Sprintf("`%s`", managedClusterName) + `
### :bulb: - follow progress with ` + fmt.Sprintf("`%s`", "kubefirst launch cluster list") + `
`
	progress.Success(deleteMessage)
}
