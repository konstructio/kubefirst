package configs

import (
	"fmt"
	"github.com/caarlos0/env/v6"
	"log"
	"os"
	"runtime"
)

// Config host application configuration
// todo: some of these values can be moved to the .env
type Config struct {
	AwsProfile        string `env:"AWS_PROFILE"`
	KubefirstLogPath  string `env:"KUBEFIRST_LOG_PATH" envDefault:"logs"`
	KubectlVersion    string `env:"KUBECTL_VERSION" envDefault:"v1.20.0"`
	HomePath          string
	LocalOs           string
	LocalArchitecture string
	KubectlClientPath string
	KubeConfigPath    string

	TerraformVersion string
	TerraformPath    string

	HelmClientPath string
	HelmVersion    string

	DryRun                        bool
	SkipDeleteRegistryApplication bool
	DestroyBuckets                bool

	KubefirstVersion string
	InstallerEmail   string

	SkipGitlabTerraform bool
	SkipBaseTerraform   bool
}

func ReadConfig() *Config {
	config := Config{}

	if err := env.Parse(&config); err != nil {
		log.Println("something went wrong loading the environment variables")
		panic(err)
	}

	var err error
	config.HomePath, err = os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	config.LocalOs = runtime.GOOS
	config.LocalArchitecture = runtime.GOARCH

	config.KubectlClientPath = fmt.Sprintf("%s/.kubefirst/tools/kubectl", config.HomePath)
	config.KubeConfigPath = fmt.Sprintf("%s/.kubefirst/gitops/terraform/base/kubeconfig_kubefirst", config.HomePath)
	config.TerraformPath = fmt.Sprintf("%s/.kubefirst/tools/terraform", config.HomePath)
	config.HelmClientPath = fmt.Sprintf("%s/.kubefirst/tools/helm", config.HomePath)

	config.TerraformVersion = "1.0.11"

	// todo adopt latest helmVersion := "v3.9.0"
	config.HelmVersion = "v3.2.1"

	config.KubefirstVersion = "0.1.1"

	config.InstallerEmail = "kubefirst-bot@kubefirst.com"

	return &config
}
