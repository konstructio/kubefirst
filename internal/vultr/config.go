/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vultr

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/caarlos0/env/v6"
	"github.com/kubefirst/kubefirst/pkg"
)

const (
	CloudProvider          = "vultr"
	GithubHost             = "github.com"
	GitlabHost             = "gitlab.com"
	KubectlClientVersion   = "v1.25.7"
	LocalhostOS            = runtime.GOOS
	LocalhostArch          = runtime.GOARCH
	TerraformClientVersion = "1.3.8"
	ArgocdHelmChartVersion = "4.10.5"

	ArgocdPortForwardURL = pkg.ArgocdPortForwardURL
	VaultPortForwardURL  = pkg.VaultPortForwardURL
)

type VultrConfig struct {
	VultrToken  string `env:"VULTR_API_KEY"`
	GithubToken string `env:"GITHUB_TOKEN"`
	GitlabToken string `env:"GITLAB_TOKEN"`

	ArgoWorkflowsDir                string
	DestinationGitopsRepoHttpsURL   string
	DestinationGitopsRepoGitURL     string
	DestinationMetaphorRepoHttpsURL string
	DestinationMetaphorRepoGitURL   string
	GitopsDir                       string
	GitProvider                     string
	K1Dir                           string
	Kubeconfig                      string
	KubectlClient                   string
	KubefirstBotSSHPrivateKey       string
	KubefirstConfig                 string
	LogsDir                         string
	MetaphorDir                     string
	RegistryAppName                 string
	RegistryYaml                    string
	SSLBackupDir                    string
	TerraformClient                 string
	ToolsDir                        string
}

// GetConfig - load default values from kubefirst installer
func GetConfig(clusterName string, domainName string, gitProvider string, gitOwner string) *VultrConfig {
	config := VultrConfig{}

	// todo do we want these from envs?
	if err := env.Parse(&config); err != nil {
		log.Panicf("error reading environment variables %s", err.Error())
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err.Error())
	}

	// cGitHost describes which git host to use depending on gitProvider
	var cGitHost string
	switch gitProvider {
	case "github":
		cGitHost = GithubHost
	case "gitlab":
		cGitHost = GitlabHost
	}

	config.DestinationGitopsRepoHttpsURL = fmt.Sprintf("https://%s/%s/gitops.git", cGitHost, gitOwner)
	config.DestinationGitopsRepoGitURL = fmt.Sprintf("git@%s:%s/gitops.git", cGitHost, gitOwner)
	config.DestinationMetaphorRepoHttpsURL = fmt.Sprintf("https://%s/%s/metaphor.git", cGitHost, gitOwner)
	config.DestinationMetaphorRepoGitURL = fmt.Sprintf("git@%s:%s/metaphor.git", cGitHost, gitOwner)

	config.ArgoWorkflowsDir = fmt.Sprintf("%s/.k1/argo-workflows", homeDir)
	config.GitopsDir = fmt.Sprintf("%s/.k1/gitops", homeDir)
	config.GitProvider = gitProvider
	config.Kubeconfig = fmt.Sprintf("%s/.k1/kubeconfig", homeDir)
	config.K1Dir = fmt.Sprintf("%s/.k1", homeDir)
	config.KubectlClient = fmt.Sprintf("%s/.k1/tools/kubectl", homeDir)
	config.KubefirstConfig = fmt.Sprintf("%s/.k1/%s", homeDir, ".kubefirst")
	config.LogsDir = fmt.Sprintf("%s/.k1/logs", homeDir)
	config.MetaphorDir = fmt.Sprintf("%s/.k1/metaphor", homeDir)
	config.RegistryAppName = "registry"
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
	ClusterId                 string
	ClusterName               string
	ClusterType               string
	DomainName                string
	KubeconfigPath            string
	KubefirstStateStoreBucket string
	KubefirstTeam             string
	KubefirstVersion          string
	StateStoreBucketHostname  string

	ArgoCDIngressURL               string
	ArgoCDIngressNoHTTPSURL        string
	ArgoWorkflowsIngressURL        string
	ArgoWorkflowsIngressNoHTTPSURL string
	ArgoWorkflowsDir               string
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

	GitlabHost         string
	GitlabOwner        string
	GitlabOwnerGroupID int
	GitlabUser         string

	GitOpsRepoAtlantisWebhookURL string
	GitOpsRepoGitURL             string
	GitOpsRepoNoHTTPSURL         string

	UseTelemetry string
}

type MetaphorTokenValues struct {
	CheckoutCWFTTemplate          string
	CloudRegion                   string
	ClusterName                   string
	CommitCWFTTemplate            string
	ContainerRegistryURL          string
	DomainName                    string
	MetaphorDevelopmentIngressURL string
	MetaphorProductionIngressURL  string
	MetaphorStagingIngressURL     string
}
