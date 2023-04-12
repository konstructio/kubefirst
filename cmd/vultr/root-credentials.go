/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vultr

import (
	"fmt"

	"github.com/kubefirst/kubefirst/internal/credentials"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/vultr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getVultrRootCredentials(cmd *cobra.Command, args []string) error {
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
	_, err = credentials.EvalAuth(vultr.CloudProvider, gitProvider)
	if err != nil {
		return err
	}

	// Instantiate kubernetes client
	config := vultr.GetConfig(clusterName, domainName, gitProvider, gitOwner)

	kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

	err = credentials.ParseAuthData(kcfg.Clientset, vultr.CloudProvider, gitProvider, domainName, &opts)
	if err != nil {
		return err
	}

	return nil
}
