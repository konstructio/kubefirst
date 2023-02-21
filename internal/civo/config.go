package civo

import (
	"fmt"

	"os"
	"runtime"

	"github.com/caarlos0/env/v6"
	"github.com/rs/zerolog/log"
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
	KubefirstConfig                 string
	LogsDir                         string
	MetaphorDir                     string
	RegistryYaml                    string
	SSLBackupDir                    string
	TerraformClient                 string
	ToolsDir                        string
}

func GetConfig(clusterName string, domainName string, githubOwner string) *CivoConfig {

	config := CivoConfig{}

	// todo do we want these from envs?
	if err := env.Parse(&config); err != nil {
		log.Panic().Msgf("error reading environment variables %s", err.Error())
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Panic().Msg(err.Error())
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

const (
	ArgocdPortForwardURL   = "http://localhost:8080"
	ArgocdHelmChartVersion = "4.10.5"
	CloudProvider          = "civo"
	GitProvider            = "github"
	HelmClientVersion      = "v3.6.1"
	KubectlClientVersion   = "v1.23.15"
	LocalhostOS            = runtime.GOOS
	LocalhostArch          = runtime.GOARCH
	TerraformClientVersion = "1.3.8"
	VaultPortForwardURL    = "http://localhost:8200"
)
