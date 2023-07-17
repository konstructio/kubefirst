/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"os"

	"github.com/kubefirst/runtime/pkg/helpers"
	"github.com/kubefirst/runtime/pkg/providerConfigs"
	"github.com/kubefirst/runtime/pkg/ssl"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func backupCivoSSL(cmd *cobra.Command, args []string) error {
	helpers.DisplayLogHints()

	clusterName := viper.GetString("flags.cluster-name")
	domainName := viper.GetString("flags.domain-name")
	gitProvider := viper.GetString("flags.git-provider")
	gitProtocol := viper.GetString("flags.git-protocol")

	// Switch based on git provider, set params
	var cGitOwner string
	switch gitProvider {
	case "github":
		cGitOwner = viper.GetString("flags.github-owner")
	case "gitlab":
		cGitOwner = viper.GetString("flags.gitlab-owner")
	default:
		log.Panic().Msgf("invalid git provider option")
	}

	config := providerConfigs.GetConfig(clusterName, domainName, gitProvider, cGitOwner, gitProtocol)

	if _, err := os.Stat(config.SSLBackupDir + "/certificates"); os.IsNotExist(err) {
		// path/to/whatever does not exist
		paths := []string{config.SSLBackupDir + "/certificates", config.SSLBackupDir + "/clusterissuers", config.SSLBackupDir + "/secrets"}

		for _, path := range paths {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				log.Info().Msgf("checking path: %s", path)
				err := os.MkdirAll(path, os.ModePerm)
				if err != nil {
					log.Info().Msg("directory already exists, continuing")
				}
			}
		}
	}

	err := ssl.Backup(config.SSLBackupDir, domainNameFlag, config.K1Dir, config.Kubeconfig)
	if err != nil {
		log.Info().Msg("error backing up ssl resources")
		return err
	}
	return nil
}
