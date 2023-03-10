package k3d

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/caarlos0/env/v6"
)

const (
	ArgocdPortForwardURL   = "http://localhost:8080"
	ArgocdURL              = "https://argocd.localdev.me"
	ArgoWorkflowsURL       = "https://argo.localdev.me"
	AtlantisURL            = "https://atlantis.localdev.me"
	ChartMuseumURL         = "https://chartmuseum.localdev.me"
	CloudProvider          = "k3d"
	DomainName             = "localdev.me"
	HelmVersion            = "v3.6.1"
	GithubHost             = "github.com"
	GitlabHost             = "gitlab.com"
	K3dVersion             = "v5.4.6"
	KubectlVersion         = "v1.22.0"
	KubefirstConsoleURL    = "https://kubefirst.localdev.me"
	LocalhostARCH          = runtime.GOARCH
	LocalhostOS            = runtime.GOOS
	MetaphorDevelopmentURL = "https://metaphor-devlopment.localdev.me"
	MetaphorStagingURL     = "https://metaphor-staging.localdev.me"
	MetaphorProductionURL  = "https://metaphor-production.localdev.me"
	MkCertVersion          = "v1.4.4"
	TerraformVersion       = "1.3.8"
	VaultPortForwardURL    = "http://localhost:8200"
	VaultURL               = "https://vault.localdev.me"
)

type K3dConfig struct {
	GithubToken string `env:"GITHUB_TOKEN"`
	CivoToken   string `env:"CIVO_TOKEN"`

	DestinationGitopsRepoGitURL   string
	DestinationMetaphorRepoGitURL string
	GitopsDir                     string
	GitProvider                   string
	HelmClient                    string
	K1Dir                         string
	K3dClient                     string
	Kubeconfig                    string
	KubectlClient                 string
	KubefirstConfig               string
	MetaphorDir                   string
	MkCertClient                  string
	MkCertPemDir                  string
	MkCertSSLSecretDir            string
	TerraformClient               string
	ToolsDir                      string
}

// GetConfig - load default values from kubefirst installer
func GetConfig(gitProvider string, gitOwner string) *K3dConfig {
	config := K3dConfig{}

	if err := env.Parse(&config); err != nil {
		log.Println("something went wrong loading the environment variables")
		log.Panic(err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}

	// cGitHost describes which git host to use depending on gitProvider
	var cGitHost string
	switch gitProvider {
	case "github":
		cGitHost = GithubHost
	case "gitlab":
		cGitHost = GitlabHost
	}

	config.DestinationGitopsRepoGitURL = fmt.Sprintf("git@%s:%s/gitops.git", cGitHost, gitOwner)
	config.DestinationMetaphorRepoGitURL = fmt.Sprintf("git@%s:%s/metaphor.git", cGitHost, gitOwner)
	config.GitopsDir = fmt.Sprintf("%s/.k1/gitops", homeDir)
	config.GitProvider = gitProvider
	config.HelmClient = fmt.Sprintf("%s/.k1/tools/helm", homeDir)
	config.K1Dir = fmt.Sprintf("%s/.k1", homeDir)
	config.K3dClient = fmt.Sprintf("%s/.k1/tools/k3d", homeDir)
	config.KubectlClient = fmt.Sprintf("%s/.k1/tools/kubectl", homeDir)
	config.Kubeconfig = fmt.Sprintf("%s/.k1/kubeconfig", homeDir)
	config.KubefirstConfig = fmt.Sprintf("%s/.kubefirst", homeDir)
	config.MetaphorDir = fmt.Sprintf("%s/.k1/metaphor", homeDir)
	config.MkCertClient = fmt.Sprintf("%s/.k1/tools/mkcert", homeDir)
	config.MkCertPemDir = fmt.Sprintf("%s/.k1/ssl/%s/pem", homeDir, DomainName)
	config.MkCertSSLSecretDir = fmt.Sprintf("%s/.k1/ssl/%s/secrets", homeDir, DomainName)
	config.TerraformClient = fmt.Sprintf("%s/.k1/tools/terraform", homeDir)
	config.ToolsDir = fmt.Sprintf("%s/.k1/tools", homeDir)

	return &config
}

type GitopsTokenValues struct {
	GithubOwner                   string
	GithubUser                    string
	GitlabOwner                   string
	GitlabOwnerGroupID            int
	GitlabUser                    string
	GitopsRepoGitURL              string
	DomainName                    string
	AtlantisAllowList             string
	AlertsEmail                   string
	ClusterName                   string
	ClusterType                   string
	GithubHost                    string
	GitlabHost                    string
	ArgoWorkflowsIngressURL       string
	VaultIngressURL               string
	ArgocdIngressURL              string
	AtlantisIngressURL            string
	MetaphorDevelopmentIngressURL string
	MetaphorStagingIngressURL     string
	MetaphorProductionIngressURL  string
	KubefirstVersion              string
	KubefirstTeam                 string
	UseTelemetry                  string
	GitProvider                   string
	CloudProvider                 string
	ClusterId                     string
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
