package configs

import (
	"fmt"
	"log"
	"os"

	"github.com/caarlos0/env/v6"
)

type CivoConfig struct {

	// environment variables
	// should we evaluate using CLUSTER_NAME and KUBEFIRST_LOG_PATH or just give them values?
	CivoToken        string `env:"CIVO_TOKEN"`
	ClusterName      string `env:"CLUSTER_NAME"`
	GithubToken      string `env:"GITHUB_TOKEN"`
	KubefirstLogPath string `env:"KUBEFIRST_LOG_PATH" envDefault:"logs"`

	// platform tool configurations
	ArgodLocalURL string
	VaultLocalURL string

	// kubefirst cli config
	GitOpsRepoPath          string
	HomePath                string
	InstallerEmail          string
	LocalArchitecture       string
	LocalOs                 string
	K1FolderPath            string
	K1ToolsPath             string
	KubeConfigFolder        string
	KubeConfigPath          string
	KubefirstConfigFileName string
	KubefirstConfigFilePath string
}

func GetCivoConfig() *CivoConfig {
	config := CivoConfig{}

	if err := env.Parse(&config); err != nil {
		log.Println("something went wrong loading the environment variables")
		log.Panic(err)
	}

	homePath, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}

	// platform tool values
	config.ArgodLocalURL = "http://localhost:8080"
	config.VaultLocalURL = "http://localhost:8200"

	// kubefirst cli config values
	config.K1FolderPath = fmt.Sprintf("%s/.k1", homePath)
	config.K1ToolsPath = fmt.Sprintf("%s/tools", config.K1FolderPath)
	config.KubeConfigPath = fmt.Sprintf("%s/kubeconfig", config.K1FolderPath)
	config.KubefirstConfigFilePath = fmt.Sprintf("%s/%s", homePath, ".kubefirst")
	config.GitOpsRepoPath = fmt.Sprintf("%s/gitops", config.K1FolderPath)

	return &config
}
