/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/caarlos0/env/v6"
)

const (
	ArgocdPortForwardURL = "http://localhost:8080"
	CloudProvider        = "k3d"
	DomainName           = "kubefirst.dev"
	GithubHost           = "github.com"
	GitlabHost           = "gitlab.com"
	K3dVersion           = "v5.4.6"
	KubectlVersion       = "v1.25.7"
	LocalhostARCH        = runtime.GOARCH
	LocalhostOS          = runtime.GOOS
	MkCertVersion        = "v1.4.4"
	TerraformVersion     = "1.3.8"
	VaultPortForwardURL  = "http://localhost:8200"
)

var (
	ArgocdURL              = fmt.Sprintf("https://argocd.%s", DomainName)
	ArgoWorkflowsURL       = fmt.Sprintf("https://argo.%s", DomainName)
	AtlantisURL            = fmt.Sprintf("https://atlantis.%s", DomainName)
	ChartMuseumURL         = fmt.Sprintf("https://chartmuseum.%s", DomainName)
	KubefirstConsoleURL    = fmt.Sprintf("https://kubefirst.%s", DomainName)
	MetaphorDevelopmentURL = fmt.Sprintf("https://metaphor-devlopment.%s", DomainName)
	MetaphorStagingURL     = fmt.Sprintf("https://metaphor-staging.%s", DomainName)
	MetaphorProductionURL  = fmt.Sprintf("https://metaphor-production.%s", DomainName)
	VaultURL               = fmt.Sprintf("https://vault.%s", DomainName)
)

type K3dConfig struct {
	GithubToken string `env:"GITHUB_TOKEN"`
	CivoToken   string `env:"CIVO_TOKEN"`

	DestinationGitopsRepoHttpsURL   string
	DestinationGitopsRepoGitURL     string
	DestinationMetaphorRepoHttpsURL string
	DestinationMetaphorRepoGitURL   string
	GitopsDir                       string
	GitProvider                     string
	K1Dir                           string
	K3dClient                       string
	Kubeconfig                      string
	KubectlClient                   string
	KubefirstConfig                 string
	MetaphorDir                     string
	MkCertClient                    string
	MkCertPemDir                    string
	MkCertSSLSecretDir              string
	TerraformClient                 string
	ToolsDir                        string
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

	config.DestinationGitopsRepoHttpsURL = fmt.Sprintf("https://%s/%s/gitops.git", cGitHost, gitOwner)
	config.DestinationGitopsRepoGitURL = fmt.Sprintf("git@%s:%s/gitops.git", cGitHost, gitOwner)
	config.DestinationMetaphorRepoHttpsURL = fmt.Sprintf("https://%s/%s/metaphor.git", cGitHost, gitOwner)
	config.DestinationMetaphorRepoGitURL = fmt.Sprintf("git@%s:%s/metaphor.git", cGitHost, gitOwner)

	config.GitopsDir = fmt.Sprintf("%s/.k1/gitops", homeDir)
	config.GitProvider = gitProvider
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
	KubeconfigPath                string
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
