/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"

	"github.com/konstructio/kubefirst-api/pkg/k3d"
	"github.com/konstructio/kubefirst-api/pkg/k8s"
	utils "github.com/konstructio/kubefirst-api/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// mkCert creates a single certificate for a host for k3d
func mkCert(cmd *cobra.Command, _ []string) error {
	utils.DisplayLogHints()

	appNameFlag, err := cmd.Flags().GetString("application")
	if err != nil {
		return fmt.Errorf("failed to get application flag: %w", err)
	}

	appNamespaceFlag, err := cmd.Flags().GetString("namespace")
	if err != nil {
		return fmt.Errorf("failed to get namespace flag: %w", err)
	}

	flags := utils.GetClusterStatusFlags()
	if !flags.SetupComplete {
		return fmt.Errorf("there doesn't appear to be an active k3d cluster")
	}

	config, err := k3d.GetConfig(
		viper.GetString("flags.cluster-name"),
		flags.GitProvider,
		viper.GetString(fmt.Sprintf("flags.%s-owner", flags.GitProvider)),
		flags.GitProtocol,
	)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	kcfg, err := k8s.CreateKubeConfig(false, config.Kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubeconfig: %w", err)
	}

	log.Infof("Generating certificate for %s.%s...", appNameFlag, k3d.DomainName)

	err = k3d.GenerateSingleTLSSecret(kcfg.Clientset, *config, appNameFlag, appNamespaceFlag)
	if err != nil {
		return fmt.Errorf("error generating certificate for %s/%s: %w", appNameFlag, appNamespaceFlag, err)
	}

	log.Infof("Certificate generated. You can use it with an app by setting `tls.secretName: %s-tls` on a Traefik IngressRoute.", appNameFlag)

	return nil
}
