/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"

	"github.com/kubefirst/kubefirst-api/pkg/credentials"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type rootCredentialsOptions struct {
	copyArgoCDPasswordToClipboard, copyKbotPasswordToClipboard, copyVaultPasswordToClipboard bool
}

func NewRootCredentialCommand() *cobra.Command {
	o := &rootCredentialsOptions{}

	cmd := &cobra.Command{
		Use:   "root-credentials",
		Short: "retrieve root authentication information for platform components",
		Long:  "retrieve root authentication information for platform components",
		RunE:  o.runK3dRootCredentials,
	}

	cmd.Flags().BoolVar(&o.copyArgoCDPasswordToClipboard, "argocd", o.copyArgoCDPasswordToClipboard, "copy the argocd password to the clipboard (optional)")
	cmd.Flags().BoolVar(&o.copyKbotPasswordToClipboard, "kbot", o.copyKbotPasswordToClipboard, "copy the kbot password to the clipboard (optional)")
	cmd.Flags().BoolVar(&o.copyVaultPasswordToClipboard, "vault", o.copyVaultPasswordToClipboard, "copy the vault password to the clipboard (optional)")

	// flag constraints
	cmd.MarkFlagsMutuallyExclusive("argocd", "kbot", "vault")

	return cmd
}

func (o *rootCredentialsOptions) runK3dRootCredentials(cmd *cobra.Command, args []string) error {
	domainName := k3d.DomainName
	clusterName := viper.GetString("flags.cluster-name")
	gitProvider := viper.GetString("flags.git-provider")
	gitProtocol := viper.GetString("flags.git-protocol")
	gitOwner := viper.GetString(fmt.Sprintf("flags.%s-owner", gitProvider))

	opts := credentials.CredentialOptions{
		CopyArgoCDPasswordToClipboard: o.copyArgoCDPasswordToClipboard,
		CopyKbotPasswordToClipboard:   o.copyKbotPasswordToClipboard,
		CopyVaultPasswordToClipboard:  o.copyVaultPasswordToClipboard,
	}

	// Determine if there are eligible installs
	if _, err := credentials.EvalAuth(k3d.CloudProvider, gitProvider); err != nil {
		return err
	}

	// Determine if the Kubernetes cluster is available
	if !viper.GetBool("kubefirst-checks.create-k3d-cluster") {
		return fmt.Errorf("it looks like a kubernetes cluster has not been created yet - try again")
	}

	// Instantiate kubernetes client
	config := k3d.GetConfig(clusterName, gitProvider, gitOwner, gitProtocol)

	kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

	if err := credentials.ParseAuthData(kcfg.Clientset, k3d.CloudProvider, gitProvider, domainName, &opts); err != nil {
		return err
	}

	return nil
}
