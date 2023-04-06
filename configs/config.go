/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package configs

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/caarlos0/env/v6"
)

/**
This is an initial implementation of Config. Please keep in mind we're still working to improve how we handle
environment variables and general config data.
*/

const DefaultK1Version = "development"

// K1Version is used on version command. The value is dynamically updated on build time via ldflag. Built Kubefirst
// versions will follow semver value like 1.9.0, when not using the built version, "development" is used.
var K1Version = DefaultK1Version

// Config host application configuration
// todo: some of these values can be moved to the .env
type Config struct {
	InstallerEmail    string
	LocalOs           string
	LocalArchitecture string
	HomePath          string

	ClusterName             string `env:"CLUSTER_NAME"`
	GitOpsRepoPath          string
	KubefirstLogPath        string `env:"KUBEFIRST_LOG_PATH" envDefault:"logs"`
	KubefirstConfigFileName string
	KubefirstConfigFilePath string
	K1FolderPath            string
	K1ToolsPath             string
	KubeConfigPath          string
	KubeConfigFolder        string
	GitOpsLocalRepoPath     string

	K3dPath            string
	MkCertPath         string
	MkCertPemFilesPath string

	CertsPath string

	HelmClientPath string
	HelmVersion    string

	K3dClientPath string

	KubectlVersionM1  string
	KubectlClientPath string

	NgrokVersion    string
	NgrokClientPath string

	TerraformClientPath string
	TerraformVersion    string

	// todo remove cloud specific values from generic config
	AwsRegion      string `env:"AWS_REGION"`
	HostedZoneName string `env:"HOSTED_ZONE_NAME"`
	K3dVersion     string
	MkCertVersion  string
	KubectlVersion string `env:"KUBECTL_VERSION" envDefault:"v1.22.0"`

	ArgoCDChartHelmVersion   string
	ArgoCDInitValuesYamlPath string

	ArgocdLocalURL   string
	ArgocdIngressURL string

	ArgoWorkflowsLocalURL   string
	ArgoWorkflowsIngressURL string

	AtlantisLocalURL   string
	AtlantisIngressURL string

	ChartmuseumLocalURL   string
	ChartmuseumIngressURL string

	MetaphorDevelopmentLocalURL string
	MetaphorStagingLocalURL     string
	MetaphorProductionLocalURL  string

	MetaphorFrontendDevelopmentLocalURL string
	MetaphorFrontendStagingLocalURL     string
	MetaphorFrontendProductionLocalURL  string

	MetaphorGoDevelopmentLocalURL string
	MetaphorGoStagingLocalURL     string
	MetaphorGoProductionLocalURL  string

	VaultLocalURL   string
	VaultIngressURL string

	TerraformAwsEntrypointPath    string
	TerraformGithubEntrypointPath string
	TerraformUsersEntrypointPath  string
	TerraformVaultEntrypointPath  string
	GithubToken                   string `env:"GITHUB_TOKEN"`
	CivoToken                     string `env:"CIVO_TOKEN"`

	// these
	GitopsDir                   string
	DestinationGitopsRepoGitURL string
	K1Dir                       string
}

// ReadConfig - load default values from kubefirst installer
func ReadConfig() *Config {
	config := Config{}

	if err := env.Parse(&config); err != nil {
		log.Println("something went wrong loading the environment variables")
		log.Panic(err)
	}

	homePath, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}

	config.HomePath = homePath
	config.K1FolderPath = fmt.Sprintf("%s/.k1", homePath)
	if err != nil {
		log.Panic(err)
	}

	config.GitopsDir = fmt.Sprintf("%s/.k1/gitops", homePath)
	config.K1Dir = fmt.Sprintf("%s/.k1", homePath)

	config.K1ToolsPath = fmt.Sprintf("%s/tools", config.K1FolderPath)
	config.KubefirstConfigFileName = ".kubefirst"
	config.KubefirstConfigFilePath = fmt.Sprintf("%s/%s", homePath, config.KubefirstConfigFileName)

	config.GitOpsRepoPath = fmt.Sprintf("%s/gitops", config.K1FolderPath)
	config.K1ToolsPath = fmt.Sprintf("%s/tools", config.K1FolderPath)
	config.KubeConfigFolder = fmt.Sprintf("%s/terraform/civo", config.GitOpsRepoPath) // civo cant be hardcoded anywhere
	config.KubeConfigPath = fmt.Sprintf("%s/terraform/civo/kubeconfig", config.GitOpsRepoPath)

	//! havent used anything below this
	config.CertsPath = fmt.Sprintf("%s/ssl", config.K1FolderPath)
	config.LocalOs = runtime.GOOS
	config.LocalArchitecture = runtime.GOARCH
	config.HelmClientPath = fmt.Sprintf("%s/helm", config.K1ToolsPath)
	config.K3dClientPath = fmt.Sprintf("%s/k3d", config.K1ToolsPath)
	config.KubectlClientPath = fmt.Sprintf("%s/kubectl", config.K1ToolsPath)
	config.NgrokClientPath = fmt.Sprintf("%s/ngrok", config.K1ToolsPath)
	config.TerraformClientPath = fmt.Sprintf("%s/terraform", config.K1ToolsPath)

	config.KubectlClientPath = fmt.Sprintf("%s/kubectl", config.K1ToolsPath)
	config.KubeConfigPath = fmt.Sprintf("%s/gitops/terraform/base/kubeconfig", config.K1FolderPath)
	config.KubeConfigFolder = fmt.Sprintf("%s/gitops/terraform/base", config.K1FolderPath)
	config.GitOpsLocalRepoPath = fmt.Sprintf("%s/gitops", config.K1FolderPath)
	config.GitOpsRepoPath = fmt.Sprintf("%s/gitops", config.K1FolderPath)
	config.NgrokClientPath = fmt.Sprintf("%s/ngrok", config.K1ToolsPath)
	config.TerraformClientPath = fmt.Sprintf("%s/terraform", config.K1ToolsPath)
	config.HelmClientPath = fmt.Sprintf("%s/helm", config.K1ToolsPath)
	config.K3dPath = fmt.Sprintf("%s/k3d", config.K1ToolsPath)
	config.CertsPath = fmt.Sprintf("%s/ssl", config.K1FolderPath)
	config.NgrokVersion = "v3"
	config.TerraformVersion = "1.3.8"
	config.ArgoCDChartHelmVersion = "4.10.5"
	config.ArgoCDInitValuesYamlPath = fmt.Sprintf("%s/argocd-init-values.yaml", config.K1FolderPath)
	// todo adopt latest helmVersion := "v3.9.0"
	config.HelmVersion = "v3.6.1"
	config.K3dVersion = "v5.4.6"

	//! cleanup below this line?
	config.InstallerEmail = "kbot@kubefirst.com"

	// Local Configs URL
	config.ArgoWorkflowsLocalURL = "http://localhost:2746"
	config.VaultLocalURL = "http://localhost:8200"
	config.ArgocdLocalURL = "http://localhost:8080"
	config.AtlantisLocalURL = "http://localhost:4141"
	config.ChartmuseumLocalURL = "http://localhost:8181"

	config.MetaphorDevelopmentLocalURL = "http://localhost:3000"
	config.MetaphorGoDevelopmentLocalURL = "http://localhost:5000"
	config.MetaphorFrontendDevelopmentLocalURL = "http://localhost:4000"

	config.MetaphorStagingLocalURL = "http://localhost:3001"
	config.MetaphorGoStagingLocalURL = "http://localhost:5001"
	config.MetaphorFrontendStagingLocalURL = "http://localhost:4001"

	config.MetaphorProductionLocalURL = "http://localhost:3002"
	config.MetaphorGoProductionLocalURL = "http://localhost:5002"
	config.MetaphorFrontendProductionLocalURL = "http://localhost:4002"
	config.InstallerEmail = "kbot@kubefirst.com"

	// certificates
	config.MkCertPath = fmt.Sprintf("%s/mkcert", config.K1ToolsPath)
	config.MkCertPemFilesPath = fmt.Sprintf("%s/certs/", config.K1ToolsPath)
	config.MkCertVersion = "v1.4.4"

	// If the AWS_SDK_LOAD_CONFIG environment variable is set to a truthy value the shared config file (~/.aws/config)
	// will also be loaded in addition to the shared credentials file (~/.aws/credentials).
	// AWS SDK client will take it in advance
	err = os.Setenv("AWS_SDK_LOAD_CONFIG", "1")
	if err != nil {
		log.Panicf("unable to set AWS_SDK_LOAD_CONFIG enviroment value, error is: %v", err)
	}

	return &config
}
