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
	LocalOs           string
	LocalArchitecture string
	InstallerEmail    string

	KubefirstLogPath        string `env:"KUBEFIRST_LOG_PATH" envDefault:"logs"`
	KubefirstConfigFileName string
	KubefirstConfigFilePath string
	K1FolderPath            string
	KubectlClientPath       string
	KubeConfigPath          string
	KubeConfigFolder        string
	HelmClientPath          string
	GitOpsRepoPath          string
	NgrokVersion            string
	NgrokClientPath         string
	TerraformClientPath     string
	K3dPath                 string

	HostedZoneName string `env:"HOSTED_ZONE_NAME"`
	ClusterName    string `env:"CLUSTER_NAME"`
	AwsRegion      string `env:"AWS_REGION"`

	K3dVersion       string
	KubectlVersion   string `env:"KUBECTL_VERSION" envDefault:"v1.20.0"`
	KubectlVersionM1 string
	TerraformVersion string
	HelmVersion      string

	ArgoCDChartHelmVersion   string
	ArgoCDInitValuesYamlPath string

	CertsPath string

	MetaphorTemplateURL string
	GitopsTemplateURL   string

	LocalArgoWorkflowsURL string
	LocalVaultURL         string
	LocalArgoURL          string
	LocalAtlantisURL      string
	LocalChartmuseumURL   string

	LocalMetaphorDev      string
	LocalMetaphorGoDev    string
	LocalMetaphorFrontDev string

	LocalMetaphorStaging      string
	LocalMetaphorGoStaging    string
	LocalMetaphorFrontStaging string

	LocalMetaphorProd      string
	LocalMetaphorGoProd    string
	LocalMetaphorFrontProd string

	GitHubPersonalAccessToken string `env:"KUBEFIRST_GITHUB_AUTH_TOKEN"`
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

	config.K1FolderPath = fmt.Sprintf("%s/.k1", homePath)
	if err != nil {
		log.Panic(err)
	}

	config.KubefirstConfigFileName = ".kubefirst"
	config.KubefirstConfigFilePath = fmt.Sprintf("%s/%s", homePath, config.KubefirstConfigFileName)

	config.LocalOs = runtime.GOOS
	config.LocalArchitecture = runtime.GOARCH

	config.KubectlClientPath = fmt.Sprintf("%s/tools/kubectl", config.K1FolderPath)
	config.KubeConfigPath = fmt.Sprintf("%s/gitops/terraform/base/kubeconfig", config.K1FolderPath)
	config.KubeConfigFolder = fmt.Sprintf("%s/gitops/terraform/base", config.K1FolderPath)
	config.GitOpsRepoPath = fmt.Sprintf("%s/gitops", config.K1FolderPath)
	config.NgrokClientPath = fmt.Sprintf("%s/tools/ngrok", config.K1FolderPath)
	config.TerraformClientPath = fmt.Sprintf("%s/tools/terraform", config.K1FolderPath)
	config.HelmClientPath = fmt.Sprintf("%s/tools/helm", config.K1FolderPath)
	config.K3dPath = fmt.Sprintf("%s/tools/k3d", config.K1FolderPath)
	config.CertsPath = fmt.Sprintf("%s/ssl", config.K1FolderPath)
	config.NgrokVersion = "v3"
	config.TerraformVersion = "1.0.11"
	config.ArgoCDChartHelmVersion = "4.10.5"
	config.ArgoCDInitValuesYamlPath = fmt.Sprintf("%s/argocd-init-values.yaml", config.K1FolderPath)
	// todo adopt latest helmVersion := "v3.9.0"
	config.HelmVersion = "v3.6.1"
	config.KubectlVersionM1 = "v1.21.14"
	config.K3dVersion = "v5.4.6"

	config.InstallerEmail = "kubefirst-bot@kubefirst.com"

	config.MetaphorTemplateURL = "https://github.com/kubefirst/metaphor-template.git"
	config.GitopsTemplateURL = "https://github.com/kubefirst/gitops-template-gh.git"
	// Local Configs URL
	config.LocalArgoWorkflowsURL = "http://localhost:2746"
	config.LocalVaultURL = "http://localhost:8200"
	config.LocalArgoURL = "http://localhost:8080"
	config.LocalAtlantisURL = "http://localhost:4141"
	config.LocalChartmuseumURL = "http://localhost:8181"

	config.LocalMetaphorDev = "http://localhost:3000"
	config.LocalMetaphorGoDev = "http://localhost:5000"
	config.LocalMetaphorFrontDev = "http://localhost:4000"

	config.LocalMetaphorStaging = "http://localhost:3001"
	config.LocalMetaphorGoStaging = "http://localhost:5001"
	config.LocalMetaphorFrontStaging = "http://localhost:4001"

	config.LocalMetaphorProd = "http://localhost:3002"
	config.LocalMetaphorGoProd = "http://localhost:5002"
	config.LocalMetaphorFrontProd = "http://localhost:4002"

	// If the AWS_SDK_LOAD_CONFIG environment variable is set to a truthy value the shared config file (~/.aws/config)
	// will also be loaded in addition to the shared credentials file (~/.aws/credentials).
	// AWS SDK client will take it in advance
	err = os.Setenv("AWS_SDK_LOAD_CONFIG", "1")
	if err != nil {
		log.Panicf("unable to set AWS_SDK_LOAD_CONFIG enviroment value, error is: %v", err)
	}

	return &config
}
