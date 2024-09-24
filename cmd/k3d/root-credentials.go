/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"errors"
	"fmt"

	"github.com/konstructio/kubefirst-api/pkg/credentials"
	"github.com/konstructio/kubefirst-api/pkg/k3d"
	"github.com/konstructio/kubefirst-api/pkg/k8s"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getK3dRootCredentials(cmd *cobra.Command, _ []string) error {
	domainName := k3d.DomainName
	clusterName := viper.GetString("flags.cluster-name")
	gitProvider := viper.GetString("flags.git-provider")
	gitProtocol := viper.GetString("flags.git-protocol")
	gitOwner := viper.GetString(fmt.Sprintf("flags.%s-owner", gitProvider))

	// Parse flags
	a, err := cmd.Flags().GetBool("argocd")
	if err != nil {
		return fmt.Errorf("failed to get ArgoCD flag: %w", err)
	}
	k, err := cmd.Flags().GetBool("kbot")
	if err != nil {
		return fmt.Errorf("failed to get kbot flag: %w", err)
	}
	v, err := cmd.Flags().GetBool("vault")
	if err != nil {
		return fmt.Errorf("failed to get vault flag: %w", err)
	}
	opts := credentials.CredentialOptions{
		CopyArgoCDPasswordToClipboard: a,
		CopyKbotPasswordToClipboard:   k,
		CopyVaultPasswordToClipboard:  v,
	}

	// Determine if there are eligible installs
	_, err = credentials.EvalAuth(k3d.CloudProvider, gitProvider)
	if err != nil {
		return fmt.Errorf("failed to evaluate auth: %w", err)
	}

	// Determine if the Kubernetes cluster is available
	if !viper.GetBool("kubefirst-checks.create-k3d-cluster") {
		return errors.New("it looks like a Kubernetes cluster has not been created yet - try again")
	}

	// Instantiate kubernetes client
	config, err := k3d.GetConfig(clusterName, gitProvider, gitOwner, gitProtocol)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	kcfg, err := k8s.CreateKubeConfig(false, config.Kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubeconfig: %w", err)
	}

	err = credentials.ParseAuthData(kcfg.Clientset, k3d.CloudProvider, domainName, &opts)
	if err != nil {
		return fmt.Errorf("failed to parse auth data: %w", err)
	}

	progress.Progress.Quit()
	return nil
}
