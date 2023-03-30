package civo

import (
	"fmt"

	"github.com/kubefirst/kubefirst/internal/civo"
	"github.com/kubefirst/kubefirst/internal/helpers"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getCivoRootCredentials(cmd *cobra.Command, args []string) error {
	clusterName := viper.GetString("flags.cluster-name")
	domainName := viper.GetString("flags.domain-name")
	gitProvider := viper.GetString("flags.git-provider")
	gitOwner := viper.GetString(fmt.Sprintf("flags.%s-owner", gitProvider))

	// Determine if there are active installs
	_, err := helpers.EvalAuth(civo.CloudProvider, gitProvider)
	if err != nil {
		return err
	}

	// Instantiate kubernetes client
	config := civo.GetConfig(clusterName, domainName, gitProvider, gitOwner)
	clientset, err := k8s.GetClientSet(false, config.Kubeconfig)
	if err != nil {
		return err
	}

	err = helpers.ParseAuthData(clientset, civo.CloudProvider, gitProvider)
	if err != nil {
		return err
	}

	return nil
}
