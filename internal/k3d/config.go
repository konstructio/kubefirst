package k3d

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/caarlos0/env/v6"
)

const (
	CloudProvider        = "k3d"
	DomainName           = "localdev.me"
	HelmVersion          = "v3.6.1"
	GitProvider          = "github"
	GithubHost           = "github.com"
	K3dVersion           = "v5.4.6"
	KubectlVersion       = "v1.22.0"
	LocalhostOS          = runtime.GOOS
	LocalhostARCH        = runtime.GOARCH
	MkCertVersion        = "v1.4.4"
	TerraformVersion     = "1.3.8"
	ArgocdPortForwardURL = "http://localhost:8080"
	VaultPortForwardURL  = "http://localhost:8200"
)

type K3dConfig struct {
	GithubToken string `env:"GITHUB_TOKEN"`
	CivoToken   string `env:"CIVO_TOKEN"`

	// these
	DestinationGitopsRepoGitURL   string
	DestinationMetaphorRepoGitURL string
	GitopsDir                     string
	MetaphorDir                   string
	HelmClient                    string
	K1Dir                         string
	K3dClient                     string
	KubectlClient                 string
	Kubeconfig                    string
	MkCertClient                  string
	TerraformClient               string
	ToolsDir                      string
}

// GetConfig - load default values from kubefirst installer
func GetConfig(githubOwner string) *K3dConfig {
	config := K3dConfig{}

	if err := env.Parse(&config); err != nil {
		log.Println("something went wrong loading the environment variables")
		log.Panic(err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}

	config.DestinationGitopsRepoGitURL = fmt.Sprintf("git@github.com:%s/gitops.git", githubOwner)
	config.DestinationMetaphorRepoGitURL = fmt.Sprintf("git@github.com:%s/metaphor-frontend.git", githubOwner)
	config.GitopsDir = fmt.Sprintf("%s/.k1/gitops", homeDir)
	config.HelmClient = fmt.Sprintf("%s/.k1/tools/helm", homeDir)
	config.K1Dir = fmt.Sprintf("%s/.k1", homeDir)
	config.K3dClient = fmt.Sprintf("%s/.k1/tools/k3d", homeDir)
	config.KubectlClient = fmt.Sprintf("%s/.k1/tools/kubectl", homeDir)
	config.Kubeconfig = fmt.Sprintf("%s/.k1/kubeconfig", homeDir)
	config.MetaphorDir = fmt.Sprintf("%s/.k1/metaphor-frontend", homeDir)
	config.MkCertClient = fmt.Sprintf("%s/.k1/tools/mkcert", homeDir)
	config.TerraformClient = fmt.Sprintf("%s/.k1/tools/terraform", homeDir)
	config.ToolsDir = fmt.Sprintf("%s/.k1/tools", homeDir)

	return &config
}

type GitopsTokenValues struct {
	GithubOwner                   string
	GithubUser                    string
	GitopsRepoGitURL              string
	DomainName                    string
	AtlantisAllowList             string
	NgrokHost                     string
	AlertsEmail                   string
	ClusterName                   string
	ClusterType                   string
	GithubHost                    string
	ArgoWorkflowsIngressURL       string
	VaultIngressURL               string
	ArgocdIngressURL              string
	AtlantisIngressURL            string
	MetaphorDevelopmentIngressURL string
	MetaphorStagingIngressURL     string
	MetaphorProductionIngressURL  string
	KubefirstVersion              string
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
