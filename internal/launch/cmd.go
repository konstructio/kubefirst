/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package launch

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kubefirst/kubefirst/internal/helm"
	k3dint "github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/runtime/configs"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/downloadManager"
	"github.com/kubefirst/runtime/pkg/helpers"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/kubefirst/runtime/pkg/k8s"
	log "github.com/sirupsen/logrus"
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
func Up(additionalHelmFlags []string) {
	if viper.GetBool("launch.deployed") {
		fmt.Println("Kubefirst console has already been deployed. To start over, run `kubefirst launch down` to completely remove the existing console.")
		os.Exit(1)
	}

	helpers.DisplayLogHints()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("something went wrong getting home path: %s", err)
	}
	dir := fmt.Sprintf("%s/.k1/%s", homeDir, consoleClusterName)
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

	if err := setupLaunchConfigFile(dir); err != nil {
		log.Fatal(err)
	}

	dbInitialized := viper.GetBool("launch.database-initialized")
	var dbHost, dbUser, dbPassword string

	if !dbInitialized {
		dbDestination := k3dint.MongoDestinationChooser()
		switch dbDestination {
		case "atlas":
			fmt.Println("MongoDB Atlas Host String: ")
			fmt.Scanln(&dbHost)

			fmt.Printf("\nMongoDB Atlas Username: ")
			fmt.Scanln(&dbUser)

			fmt.Printf("\nMongoDB Atlas Password: ")
			dbPasswordInput, err := term.ReadPassword(0)
			if err != nil {
				log.Fatalf("error parsing password: %s", err)
			}

			dbPassword = string(dbPasswordInput)
			dbHost = strings.Replace(dbHost, "mongodb+srv://", "", -1)

			fmt.Println()

			viper.Set("launch.database-destination", "atlas")
			viper.Set("launch.database-initialized", true)
			viper.WriteConfig()
		case "in-cluster":
			viper.Set("launch.database-destination", "in-cluster")
			viper.Set("launch.database-initialized", true)
			viper.WriteConfig()
		default:
			log.Fatalf("%s is not a valid option", dbDestination)
		}
	} else {
		log.Info("Database has already been initialized, skipping")
	}

	log.Infof("%s/%s", k3d.LocalhostOS, k3d.LocalhostARCH)

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
	kubeconfigPath := fmt.Sprintf("%s/.k1/%s/kubeconfig", homeDir, consoleClusterName)
	_, _, err = pkg.ExecShellReturnStrings(
		k3dClient,
		"cluster",
		"get",
		consoleClusterName,
	)
	if err != nil {
		log.Warn("k3d cluster does not exist and will be created")
		log.Info("Creating k3d cluster for Kubefirst console and API...")
		err = k3d.ClusterCreateConsoleAPI(
			consoleClusterName,
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
	} else {
		log.Warn("Kubefirst console has already been deployed. To start over, run `kubefirst launch down` to completely remove the existing console.")
		log.Warnf("If you have manually removed %s, the k3d cluster must be manually removed by running the following command: ", dir)
		log.Info("	k3d cluster delete kubefirst-console")
		log.Warn("You will have to install the k3d utility if you do not have it installed if the directory shown above has been deleted.")
		os.Exit(1)
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
			"kubefirst/kubefirst",
			"--set",
			"console.ingress.createTraefikRoute=true",
			"--set",
			fmt.Sprintf("console.chartVersion=%s", configs.K1Version),
			"--set",
			"kubefirst-api.installMethod=kubefirst-launch",
			"--set",
			fmt.Sprintf("kubefirst-api.kubefirstTeam=%s", kubefirstTeam),
			"--set",
			fmt.Sprintf("kubefirst-api.kubefirstTeamInfo=%s", kubefirstTeamInfo),
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
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, apiDeployment, 300)
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

	log.Info("To remove Kubefirst Console and the k3d cluster it runs in, please run the following command: ")
	log.Info("	kubefirst launch down")

	err = pkg.OpenBrowser(consoleURL)
	if err != nil {
		log.Errorf("error attempting to open console in browser: %s", err)
	}

	viper.Set("launch.deployed", true)
	viper.WriteConfig()
}

// Down destroys a k3d cluster for Kubefirst console and API
func Down() {
	helpers.DisplayLogHints()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("something went wrong getting home path: %s", err)
	}

	log.Info("Deleting k3d cluster for Kubefirst console and API")

	dir := fmt.Sprintf("%s/.k1/%s", homeDir, consoleClusterName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Fatalf("cluster %s directory does not exist", dir)
	}
	toolsDir := fmt.Sprintf("%s/tools", dir)
	k3dClient := fmt.Sprintf("%s/k3d", toolsDir)

	_, _, err = pkg.ExecShellReturnStrings(k3dClient, "cluster", "delete", consoleClusterName)
	if err != nil {
		log.Fatalf("error deleting k3d cluster: %s", err)
	}

	log.Info("k3d cluster for Kubefirst console and API deleted successfully")

	log.Infof("Deleting cluster directory at %s", dir)
	err = os.RemoveAll(dir)
	if err != nil {
		log.Warnf("unable to remove directory at %s", dir)
	}
}

// ListClusters makes a request to the console API to list created clusters
func ListClusters() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("something went wrong getting home path: %s", err)
	}

	dir := fmt.Sprintf("%s/.k1/%s", homeDir, consoleClusterName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Infof("unable to list clusters - cluster %s directory does not exist", dir)
	}

	// Port forward to API
	kubeconfigPath := fmt.Sprintf("%s/.k1/%s/kubeconfig", homeDir, consoleClusterName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Info("unable to list clusters - kubeconfig file does not exist")
	}

	kcfg := k8s.CreateKubeConfig(false, kubeconfigPath)
	pods, err := kcfg.Clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=kubefirst-api",
		Limit:         1,
	})
	if err != nil {
		log.Fatalf("could not find api pod: %s", err)
	}

	randPort := rand.Intn(65535-65000) + 65000
	apiStopChannel := make(chan struct{}, 1)
	defer func() {
		close(apiStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		kcfg.Clientset,
		kcfg.RestConfig,
		pods.Items[0].ObjectMeta.Name,
		"kubefirst",
		8081,
		randPort,
		apiStopChannel,
	)

	// Get lister of clusters from API
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%v/api/v1/cluster", randPort), nil)
	if err != nil {
		log.Fatalf("error creating request to api: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, getErr := httpClient.Do(req)
	if getErr != nil {
		log.Fatalf("error during api get call: %s", getErr)
	}
	if res.Body != nil {
		defer res.Body.Close()
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("error reading api response: %s", err)
	}

	var objMap []map[string]interface{}
	if err := json.Unmarshal(body, &objMap); err != nil {
		log.Fatalf("error unmarshaling api response: %s", err)
	}

	err = displayFormattedClusterInfo(objMap)
	if err != nil {
		log.Fatalf("error printing cluster list: %s", err)
	}
}

// DeleteCluster makes a request to the console API to delete a single cluster
func DeleteCluster(managedClusterName string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("something went wrong getting home path: %s", err)
	}

	dir := fmt.Sprintf("%s/.k1/%s", homeDir, consoleClusterName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Infof("unable to delete cluster - cluster %s directory does not exist", dir)
	}

	// Port forward to API
	kubeconfigPath := fmt.Sprintf("%s/.k1/%s/kubeconfig", homeDir, consoleClusterName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Info("unable to delete cluster - kubeconfig file does not exist")
	}

	kcfg := k8s.CreateKubeConfig(false, kubeconfigPath)
	pods, err := kcfg.Clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=kubefirst-api",
		Limit:         1,
	})
	if err != nil {
		log.Fatalf("could not find api pod: %s", err)
	}

	randPort := rand.Intn(65535-65000) + 65000
	apiStopChannel := make(chan struct{}, 1)
	defer func() {
		close(apiStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		kcfg.Clientset,
		kcfg.RestConfig,
		pods.Items[0].ObjectMeta.Name,
		"kubefirst",
		8081,
		randPort,
		apiStopChannel,
	)

	// Delete cluster
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://localhost:%v/api/v1/cluster/%s", randPort, managedClusterName), nil)
	if err != nil {
		log.Fatalf("error creating request to api: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, getErr := httpClient.Do(req)
	if getErr != nil {
		log.Fatalf("error during api delete call: %s", getErr)
	}
	if res.Body != nil {
		defer res.Body.Close()
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("error reading api response: %s", err)
	}

	var objMap map[string]interface{}
	if err := json.Unmarshal(body, &objMap); err != nil {
		log.Fatalf("error unmarshaling api response: %s", err)
	}

	if objMap["error"] != nil {
		fmt.Printf("error: cluster %s not found\n", managedClusterName)
		os.Exit(0)
	}

	fmt.Printf("Submitted request to delete cluster %s: %s - follow progress with `kubefirst launch list-clusters`", managedClusterName, objMap["message"])
}
