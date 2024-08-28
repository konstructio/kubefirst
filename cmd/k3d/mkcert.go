/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"

	"github.com/kubefirst/kubefirst-api/pkg/k3d"
	"github.com/kubefirst/kubefirst-api/pkg/k8s"
	utils "github.com/kubefirst/kubefirst-api/pkg/utils"
	"github.com/kubefirst/kubefirst/internal/common"
	"github.com/kubefirst/kubefirst/internal/progress"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// mkCert creates a single certificate for a host for k3d
func mkCert(cmd *cobra.Command, args []string) error {
	utils.DisplayLogHints()

	appNameFlag, err := cmd.Flags().GetString("application")
	if err != nil {
		return err
	}

	appNamespaceFlag, err := cmd.Flags().GetString("namespace")
	if err != nil {
		return err
	}

	gitopsRepoName, metaphorRepoName, err := common.GetGitmeta(viper.GetString("flags.cluster-name"))

	if err != nil {
		return fmt.Errorf("error in getting repo info: %w", err)
	}

	flags := utils.GetClusterStatusFlags()
	if !flags.SetupComplete {
		return fmt.Errorf("there doesn't appear to be an active k3d cluster")
	}
	config := k3d.GetConfig(
		viper.GetString("flags.cluster-name"),
		flags.GitProvider,
		viper.GetString(fmt.Sprintf("flags.%s-owner", flags.GitProvider)),
		flags.GitProtocol,
		gitopsRepoName,
		metaphorRepoName,
		viper.GetString("adminTeamName"),
		viper.GetString("developerTeamName"),
	)
	kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

	log.Infof("Generating certificate for %s.%s...", appNameFlag, k3d.DomainName)

	err = k3d.GenerateSingleTLSSecret(kcfg.Clientset, *config, appNameFlag, appNamespaceFlag)
	if err != nil {
		return fmt.Errorf("error generating certificate for %s/%s: %w", appNameFlag, appNamespaceFlag, err)
	}

	log.Infof("Certificate generated. You can use it with an app by setting `tls.secretName: %s-tls` on a Traefik IngressRoute.", appNameFlag)
	progress.Progress.Quit()

	return nil
}
