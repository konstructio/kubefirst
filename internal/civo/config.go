package civo

import (
	"fmt"
	"log"
	"runtime"

	"github.com/caarlos0/env/v6"
)

// todo can we take the final struct

type CivoConfig struct {
	CivoToken        string `env:"CIVO_TOKEN"`
	ClusterName      string `env:"CLUSTER_NAME"`
	GithubToken      string `env:"GITHUB_TOKEN"`
	KubefirstLogPath string `env:"KUBEFIRST_LOG_PATH" envDefault:"logs"`

	DestinationGitopsRepoHttpsURL   string
	DestinationGitopsRepoGitURL     string
	DestinationMetaphorRepoHttpsURL string
	DestinationMetaphorRepoGitURL   string

	GitopsDir       string
	HelmClient      string
	Kubeconfig      string
	kubectlClient   string
	KubefirstConfig string
	LogsDir         string
	MetaphorDir     string
	TerraformClient string
	ToolsDir        string
}

func GetConfig(homeDir, githubOwnerFlag string) *CivoConfig {

	config := CivoConfig{}

	// todo do we want these from envs?
	if err := env.Parse(&config); err != nil {
		log.Println("something went wrong loading the environment variables")
		log.Panic(err)
	}

	config.DestinationGitopsRepoHttpsURL = fmt.Sprintf("https://github.com/%s/gitops.git", githubOwnerFlag)
	config.DestinationGitopsRepoGitURL = fmt.Sprintf("git@github.com:%s/gitops.git", githubOwnerFlag)
	config.DestinationMetaphorRepoHttpsURL = fmt.Sprintf("https://github.com/%s/metaphor-frontend.git", githubOwnerFlag)
	config.DestinationMetaphorRepoGitURL = fmt.Sprintf("git@github.com:%s/metaphor-frontend.git", githubOwnerFlag)

	config.GitopsDir = fmt.Sprintf("%s/.k1/gitops", homeDir)
	config.HelmClient = fmt.Sprintf("%s/.k1/tools/helm", homeDir)
	config.Kubeconfig = fmt.Sprintf("%s/.k1/kubeconfig", homeDir)
	config.kubectlClient = fmt.Sprintf("%s/.k1/tools/kubectl", homeDir)
	config.KubefirstConfig = fmt.Sprintf("%s/.k1/%s", homeDir, ".kubefirst")
	config.LogsDir = fmt.Sprintf("%s/.k1/logs", homeDir)
	config.MetaphorDir = fmt.Sprintf("%s/.k1/metaphor-frontend", homeDir)
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
