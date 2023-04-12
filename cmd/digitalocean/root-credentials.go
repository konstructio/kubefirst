/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package digitalocean

import (
	"fmt"

	"github.com/kubefirst/runtime/pkg/credentials"
	"github.com/kubefirst/runtime/pkg/digitalocean"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getDigitaloceanRootCredentials(cmd *cobra.Command, args []string) error {
	clusterName := viper.GetString("flags.cluster-name")
	domainName := viper.GetString("flags.domain-name")
	gitProvider := viper.GetString("flags.git-provider")
	gitOwner := viper.GetString(fmt.Sprintf("flags.%s-owner", gitProvider))

	// Parse flags
	a, err := cmd.Flags().GetBool("argocd")
	if err != nil {
		return err
	}
	k, err := cmd.Flags().GetBool("kbot")
	if err != nil {
		return err
	}
	v, err := cmd.Flags().GetBool("vault")
	if err != nil {
		return err
	}
	opts := credentials.CredentialOptions{
		CopyArgoCDPasswordToClipboard: a,
		CopyKbotPasswordToClipboard:   k,
		CopyVaultPasswordToClipboard:  v,
	}

	// Determine if there are active installs
	_, err = credentials.EvalAuth(digitalocean.CloudProvider, gitProvider)
	if err != nil {
		return err
	}

	// Instantiate kubernetes client
	config := digitalocean.GetConfig(clusterName, domainName, gitProvider, gitOwner)

	kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

	err = credentials.ParseAuthData(kcfg.Clientset, digitalocean.CloudProvider, gitProvider, domainName, &opts)
	if err != nil {
		return err
	}

	return nil
}
