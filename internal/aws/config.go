package aws

import (
	"fmt"
	"os"

	"github.com/caarlos0/env/v6"
	"github.com/rs/zerolog/log"
)

// todo move shared constants to pkg.
const (
	ArgocdPortForwardURL   = "http://localhost:8080"
	ArgocdURL              = "https://argocd.localdev.me"
	ArgoWorkflowsURL       = "https://argo.localdev.me"
	AtlantisURL            = "https://atlantis.localdev.me"
	ChartMuseumURL         = "https://chartmuseum.localdev.me"
	RegionUsEast1          = "us-east-1"
	CloudProvider          = "aws"
	DomainName             = "localdev.me"
	HelmVersion            = "v3.6.1"
	GithubHost             = "github.com"
	GitProvider            = "github"
	KubectlVersion         = "v1.22.0"
	KubefirstConsoleURL    = "https://kubefirst.localdev.me"
	MetaphorDevelopmentURL = "https://metaphor-devlopment.localdev.me"
	MetaphorStagingURL     = "https://metaphor-staging.localdev.me"
	MetaphorProductionURL  = "https://metaphor-production.localdev.me"
	MkCertVersion          = "v1.4.4"
	TerraformVersion       = "1.3.8"
	VaultPortForwardURL    = "http://localhost:8200"
	VaultURL               = "https://vault.localdev.me"
)

// todo standardize on field names
type GitOpsDirectoryValues struct {
	AlertsEmail                    string
	ArgoCDIngressURL               string
	ArgoCDIngressNoHTTPSURL        string
	ArgoWorkflowsIngressNoHTTPSURL string
	ArgoWorkflowsIngressURL        string
	AtlantisIngressURL             string
	AtlantisIngressNoHTTPSURL      string
	AtlantisAllowList              string
	AtlantisWebhookURL             string
	AwsIamArnAccountRoot           string
	AwsKmsKeyId                    string
	AwsNodeCapacityType            string
	ChartMuseumIngressURL          string
	ClusterName                    string
	ClusterType                    string
	CloudProvider                  string
	CloudRegion                    string
	DomainName                     string
	GithubHost                     string
	GithubOwner                    string
	GithubUser                     string
	GitDescription                 string
	GitNamespace                   string
	GitProvider                    string
	GitRunner                      string
	GitRunnerDescription           string
	GitRunnerNS                    string
	GitopsRepoGitURL               string
	Kubeconfig                     string
	GitHubHost                     string
	GitHubOwner                    string
	GitHubUser                     string
	GitOpsRepoAtlantisWebhookURL   string
	GitOpsRepoGitURL               string
	GitOpsRepoNoHTTPSURL           string
	KubefirstArtifactsBucket       string
	KubefirstStateStoreBucket      string
	KubefirstTeam                  string
	KubefirstVersion               string
	MetaphorDevelopmentIngressURL  string
	MetaphorStagingIngressURL      string
	MetaphorProductionIngressURL   string
	VaultIngressURL                string
	VaultIngressNoHTTPSURL         string
	VouchIngressURL                string
}

type MetaphorTokenValues struct {
	ClusterName                   string
	CloudRegion                   string
	ContainerRegistryURL          string
	DomainName                    string
	MetaphorDevelopmentIngressURL string
	MetaphorStagingIngressURL     string
	MetaphorProductionIngressURL  string
}

type AwsConfig struct {
	DestinationGitopsRepoGitURL   string
	DestinationMetaphorRepoGitURL string
	GitopsDir                     string
	HelmClient                    string
	K1Dir                         string
	KubectlClient                 string
	Kubeconfig                    string
	KubefirstConfig               string
	MetaphorDir                   string
	TerraformClient               string
	ToolsDir                      string
}

// todo move shared values to pkg. or break into common shared configs across git
// GetConfig - load default values from kubefirst installer
func GetConfig(githubOwner string) *AwsConfig {
	config := AwsConfig{}

	if err := env.Parse(&config); err != nil {
		log.Info().Msg("something went wrong loading the environment variables")
		log.Panic().Msg(err.Error())
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Panic().Msg(err.Error())
	}

	config.DestinationGitopsRepoGitURL = fmt.Sprintf("git@github.com:%s/gitops.git", githubOwner)
	config.DestinationMetaphorRepoGitURL = fmt.Sprintf("git@github.com:%s/metaphor.git", githubOwner)
	config.GitopsDir = fmt.Sprintf("%s/.k1/gitops", homeDir)
	config.HelmClient = fmt.Sprintf("%s/.k1/tools/helm", homeDir)
	config.K1Dir = fmt.Sprintf("%s/.k1", homeDir)
	config.KubectlClient = fmt.Sprintf("%s/.k1/tools/kubectl", homeDir)
	config.Kubeconfig = fmt.Sprintf("%s/.k1/kubeconfig", homeDir)
	config.KubefirstConfig = fmt.Sprintf("%s/.kubefirst", homeDir)
	config.MetaphorDir = fmt.Sprintf("%s/.k1/metaphor", homeDir)
	config.TerraformClient = fmt.Sprintf("%s/.k1/tools/terraform", homeDir)
	config.ToolsDir = fmt.Sprintf("%s/.k1/tools", homeDir)

	return &config
}
