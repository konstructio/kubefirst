package k3d

import (
	"fmt"

	"github.com/kubefirst/kubefirst/internal/helpers"
	"github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getK3dRootCredentials(cmd *cobra.Command, args []string) error {
	gitProvider := viper.GetString("flags.git-provider")
	gitOwner := viper.GetString(fmt.Sprintf("flags.%s-owner", gitProvider))

	// Determine if there are active installs
	_, err := helpers.EvalAuth(k3d.CloudProvider, gitProvider)
	if err != nil {
		return err
	}

	// Instantiate kubernetes client
	config := k3d.GetConfig(gitProvider, gitOwner)
	clientset, err := k8s.GetClientSet(false, config.Kubeconfig)
	if err != nil {
		return err
	}

	err = helpers.ParseAuthData(clientset, k3d.CloudProvider, gitProvider)
	if err != nil {
		return err
	}

	return nil
}
