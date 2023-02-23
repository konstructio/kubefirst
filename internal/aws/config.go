package aws

import (
	"fmt"
	"log"
	"os"

	"github.com/caarlos0/env/v6"
)

// todo move shared constants to pkg.
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
	GitProvider            = "github"
	K3dVersion             = "v5.4.6"
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

type GitopsTokenValues struct {
}

type MetaphorTokenValues struct {
}

type AwsConfig struct {
	DestinationGitopsRepoGitURL   string
	DestinationMetaphorRepoGitURL string
	GitopsDir                     string
	HelmClient                    string
	K1Dir                         string
	K3dClient                     string
	KubectlClient                 string
	Kubeconfig                    string
	KubefirstConfig               string
	MetaphorDir                   string
	MkCertClient                  string
	TerraformClient               string
	ToolsDir                      string
}

// todo move shared values to pkg. or break into common shared configs across git
// GetConfig - load default values from kubefirst installer
func GetConfig(githubOwner string) *AwsConfig {
	config := AwsConfig{}

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
	config.KubefirstConfig = fmt.Sprintf("%s/.kubefirst", homeDir)
	config.MetaphorDir = fmt.Sprintf("%s/.k1/metaphor-frontend", homeDir)
	config.MkCertClient = fmt.Sprintf("%s/.k1/tools/mkcert", homeDir)
	config.TerraformClient = fmt.Sprintf("%s/.k1/tools/terraform", homeDir)
	config.ToolsDir = fmt.Sprintf("%s/.k1/tools", homeDir)

	return &config
}
