package civo

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/caarlos0/env/v6"
)

const (
	CloudProvider          = "civo"
	GitProvider            = "github"
	GithubHost             = "github.com"
	HelmClientVersion      = "v3.6.1"
	HelmVersion            = "v3.6.1"
	KubectlClientVersion   = "v1.23.15"
	KubectlVersion         = "v1.22.0"
	LocalhostOS            = runtime.GOOS
	LocalhostArch          = runtime.GOARCH
	TerraformClientVersion = "1.3.8"
	TerraformVersion       = "1.3.8"

	ArgocdHelmChartVersion = "4.10.5"
	ArgocdPortForwardURL   = "http://localhost:8080"
	VaultPortForwardURL    = "http://localhost:8200"
)

type CivoConfig struct {
	CivoToken   string `env:"CIVO_TOKEN"`
	GithubToken string `env:"GITHUB_TOKEN"`

	DestinationGitopsRepoHttpsURL   string
	DestinationGitopsRepoGitURL     string
	DestinationMetaphorRepoHttpsURL string
	DestinationMetaphorRepoGitURL   string
	GitopsDir                       string
	HelmClient                      string
	K1Dir                           string
	Kubeconfig                      string
	KubectlClient                   string
	KubefirstBotSSHPrivateKey       string
	KubefirstConfig                 string
	LogsDir                         string
	MetaphorDir                     string
	RegistryYaml                    string
	SSLBackupDir                    string
	TerraformClient                 string
	ToolsDir                        string
}

// GetConfig - load default values from kubefirst installer
func GetConfig(clusterName string, domainName string, githubOwner string) *CivoConfig {
	config := CivoConfig{}

	// todo do we want these from envs?
	if err := env.Parse(&config); err != nil {
		log.Panic(fmt.Sprintf("error reading environment variables %s", err.Error()))
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err.Error())
	}

	config.DestinationGitopsRepoHttpsURL = fmt.Sprintf("https://github.com/%s/gitops.git", githubOwner)
	config.DestinationGitopsRepoGitURL = fmt.Sprintf("git@github.com:%s/gitops.git", githubOwner)
	config.DestinationMetaphorRepoHttpsURL = fmt.Sprintf("https://github.com/%s/metaphor-frontend.git", githubOwner)
	config.DestinationMetaphorRepoGitURL = fmt.Sprintf("git@github.com:%s/metaphor-frontend.git", githubOwner)

	config.GitopsDir = fmt.Sprintf("%s/.k1/gitops", homeDir)
	config.HelmClient = fmt.Sprintf("%s/.k1/tools/helm", homeDir)
	config.Kubeconfig = fmt.Sprintf("%s/.k1/kubeconfig", homeDir)
	config.K1Dir = fmt.Sprintf("%s/.k1", homeDir)
	config.KubectlClient = fmt.Sprintf("%s/.k1/tools/kubectl", homeDir)
	config.KubefirstConfig = fmt.Sprintf("%s/.k1/%s", homeDir, ".kubefirst")
	config.LogsDir = fmt.Sprintf("%s/.k1/logs", homeDir)
	config.MetaphorDir = fmt.Sprintf("%s/.k1/metaphor-frontend", homeDir)
	config.RegistryYaml = fmt.Sprintf("%s/.k1/gitops/registry/%s/registry.yaml", homeDir, clusterName)
	config.SSLBackupDir = fmt.Sprintf("%s/.k1/ssl/%s", homeDir, domainName)
	config.TerraformClient = fmt.Sprintf("%s/.k1/tools/terraform", homeDir)
	config.ToolsDir = fmt.Sprintf("%s/.k1/tools", homeDir)

	return &config
}

type GitOpsDirectoryValues struct {
	AlertsEmail               string
	AtlantisAllowList         string
	CloudProvider             string
	CloudRegion               string
	ClusterName               string
	ClusterType               string
	DomainName                string
	KubeconfigPath            string
	KubefirstStateStoreBucket string
	KubefirstTeam             string
	KubefirstVersion          string

	ArgoCDIngressURL               string
	ArgoCDIngressNoHTTPSURL        string
	ArgoWorkflowsIngressURL        string
	ArgoWorkflowsIngressNoHTTPSURL string
	AtlantisIngressURL             string
	AtlantisIngressNoHTTPSURL      string
	ChartMuseumIngressURL          string
	VaultIngressURL                string
	VaultIngressNoHTTPSURL         string
	VouchIngressURL                string

	GitDescription       string
	GitNamespace         string
	GitProvider          string
	GitRunner            string
	GitRunnerDescription string
	GitRunnerNS          string
	GitURL               string

	GitHubHost  string
	GitHubOwner string
	GitHubUser  string

	GitOpsRepoAtlantisWebhookURL string
	GitOpsRepoGitURL             string
	GitOpsRepoNoHTTPSURL         string

	// MetaphorDevelopmentIngressURL                string
	// MetaphorDevelopmentIngressNoHTTPSURL         string
	// MetaphorProductionIngressURL                 string
	// MetaphorProductionIngressNoHTTPSURL          string
	// MetaphorStagingIngressURL                    string
	// MetaphorStagingIngressNoHTTPSURL             string
	// MetaphorFrontendDevelopmentIngressURL        string
	// MetaphorFrontendDevelopmentIngressNoHTTPSURL string
	// MetaphorFrontendProductionIngressURL         string
	// MetaphorFrontendProductionIngressNoHTTPSURL  string
	// MetaphorFrontendStagingIngressURL            string
	// MetaphorFrontendStagingIngressNoHTTPSURL     string
	// MetaphorGoDevelopmentIngressURL              string
	// MetaphorGoDevelopmentIngressNoHTTPSURL       string
	// MetaphorGoProductionIngressURL               string
	// MetaphorGoProductionIngressNoHTTPSURL        string
	// MetaphorGoStagingIngressURL                  string
	// MetaphorGoStagingIngressNoHTTPSURL           string
}

type MetaphorTokenValues struct {
	CheckoutCWFTTemplate                  bool
	CloudRegion                           string
	ClusterName                           string
	CommitCWFTTemplate                    bool
	ContainerRegistryURL                  string
	DomainName                            string
	MetaphorFrontendDevelopmentIngressURL string
	MetaphorFrontendProductionIngressURL  string
	MetaphorFrontendStagingIngressURL     string
}
