/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"fmt"

	"github.com/kubefirst/kubefirst/internal/civo"
	"github.com/kubefirst/kubefirst/internal/credentials"
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
	_, err := credentials.EvalAuth(civo.CloudProvider, gitProvider)
	if err != nil {
		return err
	}

	// Instantiate kubernetes client
	config := civo.GetConfig(clusterName, domainName, gitProvider, gitOwner)

	kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

	err = credentials.ParseAuthData(kcfg.Clientset, civo.CloudProvider, gitProvider)
	if err != nil {
		return err
	}

	return nil
}
