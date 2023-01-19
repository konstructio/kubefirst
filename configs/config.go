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
	HomePath          string

	KubefirstLogPath        string `env:"KUBEFIRST_LOG_PATH" envDefault:"logs"`
	KubefirstConfigFileName string
	KubefirstConfigFilePath string
	K1FolderPath            string
	K1ToolsPath             string
	KubectlClientPath       string
	KubeConfigPath          string
	KubeConfigFolder        string
	HelmClientPath          string
	GitOpsLocalRepoPath     string
	GitOpsRepoPath          string

	NgrokVersion        string
	NgrokClientPath     string
	TerraformClientPath string
	K3dPath             string
	MkCertPath          string
	MkCertPemFilesPath  string

	HostedZoneName string `env:"HOSTED_ZONE_NAME"`
	ClusterName    string `env:"CLUSTER_NAME"`
	AwsRegion      string `env:"AWS_REGION"`

	K3dVersion       string
	MkCertVersion    string
	KubectlVersion   string `env:"KUBECTL_VERSION" envDefault:"v1.22.0"`
	TerraformVersion string
	HelmVersion      string

	ArgoCDChartHelmVersion   string
	ArgoCDInitValuesYamlPath string

	CertsPath string

	MetaphorTemplateURL string
	GitopsTemplateURL   string

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

	config.HomePath = homePath
	config.K1FolderPath = fmt.Sprintf("%s/.k1", homePath)
	if err != nil {
		log.Panic(err)
	}
	config.K1ToolsPath = fmt.Sprintf("%s/tools", config.K1FolderPath)
	config.KubefirstConfigFileName = ".kubefirst"
	config.KubefirstConfigFilePath = fmt.Sprintf("%s/%s", homePath, config.KubefirstConfigFileName)

	config.LocalOs = runtime.GOOS
	config.LocalArchitecture = runtime.GOARCH

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
	config.TerraformVersion = "1.0.11"
	config.ArgoCDChartHelmVersion = "4.10.5"
	config.ArgoCDInitValuesYamlPath = fmt.Sprintf("%s/argocd-init-values.yaml", config.K1FolderPath)
	// todo adopt latest helmVersion := "v3.9.0"
	config.HelmVersion = "v3.6.1"
	config.K3dVersion = "v5.4.6"
	config.InstallerEmail = "kubefirst-bot@kubefirst.com"

	// certificates
	config.MkCertPath = fmt.Sprintf("%s/mkcert", config.K1ToolsPath)
	config.MkCertPemFilesPath = fmt.Sprintf("%s/certs/", config.K1ToolsPath)
	config.MkCertVersion = "v1.4.4"

	config.MetaphorTemplateURL = "https://github.com/kubefirst/metaphor-template.git"
	config.GitopsTemplateURL = "https://github.com/kubefirst/gitops-template-gh.git"

	// If the AWS_SDK_LOAD_CONFIG environment variable is set to a truthy value the shared config file (~/.aws/config)
	// will also be loaded in addition to the shared credentials file (~/.aws/credentials).
	// AWS SDK client will take it in advance
	err = os.Setenv("AWS_SDK_LOAD_CONFIG", "1")
	if err != nil {
		log.Panicf("unable to set AWS_SDK_LOAD_CONFIG enviroment value, error is: %v", err)
	}

	return &config
}
