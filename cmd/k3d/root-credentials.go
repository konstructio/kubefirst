/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"

	"github.com/konstructio/kubefirst-api/pkg/credentials"
	"github.com/konstructio/kubefirst-api/pkg/k3d"
	"github.com/konstructio/kubefirst-api/pkg/k8s"
	"github.com/konstructio/kubefirst/internal/common"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getK3dRootCredentials(cmd *cobra.Command, args []string) error {
	domainName := k3d.DomainName
	clusterName := viper.GetString("flags.cluster-name")
	gitProvider := viper.GetString("flags.git-provider")
	gitProtocol := viper.GetString("flags.git-protocol")
	gitOwner := viper.GetString(fmt.Sprintf("flags.%s-owner", gitProvider))
	gitopsRepoName, metaphorRepoName, err := common.GetGitmeta(clusterName)

	if err != nil {
		return fmt.Errorf("error in getting repo info: %w", err)
	}

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

	// Determine if there are eligible installs
	_, err = credentials.EvalAuth(k3d.CloudProvider, gitProvider)
	if err != nil {
		return err
	}

	// Determine if the Kubernetes cluster is available
	if !viper.GetBool("kubefirst-checks.create-k3d-cluster") {
		return fmt.Errorf("it looks like a kubernetes cluster has not been created yet - try again")
	}

	// Instantiate kubernetes client
	config := k3d.GetConfig(clusterName, gitProvider, gitOwner, gitProtocol, gitopsRepoName, metaphorRepoName, viper.GetString("adminTeamName"), viper.GetString("developerTeamName"))

	kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

	err = credentials.ParseAuthData(kcfg.Clientset, k3d.CloudProvider, gitProvider, domainName, &opts)
	if err != nil {
		return err
	}

	progress.Progress.Quit()

	return nil
}
