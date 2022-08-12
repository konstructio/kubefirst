package configs

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/caarlos0/env/v6"
)

/**
This is an initial implementation of Config. Please keep in mind we're still working to improve how we handle
environment variables and general config data.
*/

// Config host application configuration
// todo: some of these values can be moved to the .env
type Config struct {
	LocalOs           string
	LocalArchitecture string
	InstallerEmail    string

	KubefirstLogPath        string `env:"KUBEFIRST_LOG_PATH" envDefault:"logs"`
	KubefirstConfigFileName string
	KubefirstConfigFilePath string
	K1FolderPath            string
	KubectlClientPath       string
	KubeConfigPath          string
	HelmClientPath          string
	TerraformPath           string

	KubectlVersion   string `env:"KUBECTL_VERSION" envDefault:"v1.20.0"`
	TerraformVersion string
	HelmVersion      string

	// todo: move it back
	KubefirstVersion       string
	ArgoCDChartHelmVersion string

	CertsPath string
}

func ReadConfig() *Config {
	config := Config{}

	if err := env.Parse(&config); err != nil {
		log.Println("something went wrong loading the environment variables")
		log.Panic(err)
	}

	homePath, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}

	config.K1FolderPath = fmt.Sprintf("%s/.k1", homePath)
	if err != nil {
		log.Panic(err)
	}

	config.KubefirstConfigFileName = ".kubefirst"
	config.KubefirstConfigFilePath = fmt.Sprintf("%s/.kubefirst", homePath)

	config.LocalOs = runtime.GOOS
	config.LocalArchitecture = runtime.GOARCH

	config.KubectlClientPath = fmt.Sprintf("%s/tools/kubectl", config.K1FolderPath)
	config.KubeConfigPath = fmt.Sprintf("%s/gitops/terraform/base/kubeconfig", config.K1FolderPath)
	config.TerraformPath = fmt.Sprintf("%s/tools/terraform", config.K1FolderPath)
	config.HelmClientPath = fmt.Sprintf("%s/tools/helm", config.K1FolderPath)
	config.CertsPath = fmt.Sprintf("%s/ssl", config.K1FolderPath)
	config.TerraformVersion = "1.0.11"
	config.ArgoCDChartHelmVersion = "4.10.5"
	// todo adopt latest helmVersion := "v3.9.0"
	config.HelmVersion = "v3.2.1"

	config.KubefirstVersion = "1.8.5"

	config.InstallerEmail = "kubefirst-bot@kubefirst.com"

	// If the AWS_SDK_LOAD_CONFIG environment variable is set to a truthy value the shared config file (~/.aws/config)
	// will also be loaded in addition to the shared credentials file (~/.aws/credentials).
	// AWS SDK client will take it in advance
	err = os.Setenv("AWS_SDK_LOAD_CONFIG", "1")
	if err != nil {
		log.Panicf("unable to set AWS_SDK_LOAD_CONFIG enviroment value, error is: %v", err)
	}

	return &config
}
