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
	InstallerEmail string

	ClusterName             string `env:"CLUSTER_NAME"`
	GitOpsRepoPath          string
	KubefirstLogPath        string `env:"KUBEFIRST_LOG_PATH" envDefault:"logs"`
	KubefirstConfigFileName string
	KubefirstConfigFilePath string
	K1FolderPath            string
	K1ToolsPath             string
	KubeConfigPath          string
	KubeConfigFolder        string

	LocalOs           string
	LocalArchitecture string

	CertsPath string

	HelmClientPath string
	HelmVersion    string

	K3dClientPath string
	K3dVersion    string

	KubectlVersion    string `env:"KUBECTL_VERSION" envDefault:"v1.20.0"`
	KubectlVersionM1  string
	KubectlClientPath string

	NgrokVersion    string
	NgrokClientPath string

	TerraformClientPath string
	TerraformVersion    string

	// todo remove cloud specific values from generic config
	AwsRegion      string `env:"AWS_REGION"`
	HostedZoneName string `env:"HOSTED_ZONE_NAME"`

	ArgoCDChartHelmVersion   string
	ArgoCDInitValuesYamlPath string

	ArgocdLocalUrl   string
	ArgocdIngressUrl string

	ArgoWorkflowsLocalUrl   string
	ArgoWorkflowsIngressUrl string

	AtlantisLocalUrl   string
	AtlantisIngressUrl string

	ChartmuseumLocalUrl   string
	ChartmuseumIngressUrl string

	MetaphorDevelopmentLocalUrl string
	MetaphorStagingLocalUrl     string
	MetaphorProductionLocalUrl  string

	MetaphorFrontendDevelopmentLocalUrl string
	MetaphorFrontendStagingLocalUrl     string
	MetaphorFrontendProductionLocalUrl  string

	MetaphorGoDevelopmentLocalUrl string
	MetaphorGoStagingLocalUrl     string
	MetaphorGoProductionLocalUrl  string

	VaultLocalUrl   string
	VaultIngressUrl string

	TerraformAwsEntrypointPath    string
	TerraformGithubEntrypointPath string
	TerraformUsersEntrypointPath  string
	TerraformVaultEntrypointPath  string
	GithubToken                   string `env:"GITHUB_TOKEN"`
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

	config.NgrokVersion = "v3"
	config.TerraformVersion = "1.0.11"
	config.ArgoCDChartHelmVersion = "4.10.5"
	config.ArgoCDInitValuesYamlPath = fmt.Sprintf("%s/argocd-init-values.yaml", config.K1FolderPath)
	// todo adopt latest helmVersion := "v3.9.0"
	config.HelmVersion = "v3.6.1"
	config.KubectlVersionM1 = "v1.21.14"
	config.K3dVersion = "v5.4.6"

	//! cleanup below this line?
	config.InstallerEmail = "kubefirst-bot@kubefirst.com"

	// Local Configs URL
	config.ArgoWorkflowsLocalUrl = "http://localhost:2746"
	config.VaultLocalUrl = "http://localhost:8200"
	config.ArgocdLocalUrl = "http://localhost:8080"
	config.AtlantisLocalUrl = "http://localhost:4141"
	config.ChartmuseumLocalUrl = "http://localhost:8181"

	config.MetaphorDevelopmentLocalUrl = "http://localhost:3000"
	config.MetaphorGoDevelopmentLocalUrl = "http://localhost:5000"
	config.MetaphorFrontendDevelopmentLocalUrl = "http://localhost:4000"

	config.MetaphorStagingLocalUrl = "http://localhost:3001"
	config.MetaphorGoStagingLocalUrl = "http://localhost:5001"
	config.MetaphorFrontendStagingLocalUrl = "http://localhost:4001"

	config.MetaphorProductionLocalUrl = "http://localhost:3002"
	config.MetaphorGoProductionLocalUrl = "http://localhost:5002"
	config.MetaphorFrontendProductionLocalUrl = "http://localhost:4002"

	// If the AWS_SDK_LOAD_CONFIG environment variable is set to a truthy value the shared config file (~/.aws/config)
	// will also be loaded in addition to the shared credentials file (~/.aws/credentials).
	// AWS SDK client will take it in advance
	err = os.Setenv("AWS_SDK_LOAD_CONFIG", "1")
	if err != nil {
		log.Panicf("unable to set AWS_SDK_LOAD_CONFIG enviroment value, error is: %v", err)
	}

	return &config
}
