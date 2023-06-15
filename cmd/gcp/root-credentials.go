/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gcp

import (
	"context"
	"fmt"

	"github.com/kubefirst/runtime/pkg/credentials"
	"github.com/kubefirst/runtime/pkg/gcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getGCPRootCredentials(cmd *cobra.Command, args []string) error {
	clusterName := viper.GetString("flags.cluster-name")
	domainName := viper.GetString("flags.domain-name")
	gitProvider := viper.GetString("flags.git-provider")
	gcpProject := viper.GetString("flags.gcp-project")

	fmt.Println(clusterName)

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
	_, err = credentials.EvalAuth(gcp.CloudProvider, gitProvider)
	if err != nil {
		return err
	}

	// Determine if the Kubernetes cluster is available
	if !viper.GetBool("kubefirst-checks.terraform-apply-gcp") {
		return fmt.Errorf("it looks like a kubernetes cluster has not been created yet - try again")
	}

	// Instantiate kubernetes client
	gcpConf := gcp.GCPConfiguration{
		Context: context.Background(),
		Project: gcpProject,
		Region:  cloudRegionFlag,
	}
	kcfg, err := gcpConf.GetContainerClusterAuth(clusterName)
	if err != nil {
		return fmt.Errorf("could not build kubernetes config for gcp cluster %s: %s", clusterName, err)
	}

	err = credentials.ParseAuthData(kcfg.Clientset, gcp.CloudProvider, gitProvider, domainName, &opts)
	if err != nil {
		return err
	}

	return nil
}
