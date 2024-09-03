/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"fmt"
	"os"

	"github.com/konstructio/kubefirst-api/pkg/providerConfigs"
	"github.com/konstructio/kubefirst-api/pkg/ssl"
	utils "github.com/konstructio/kubefirst-api/pkg/utils"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func backupCivoSSL(_ *cobra.Command, _ []string) error {
	utils.DisplayLogHints()

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
		return fmt.Errorf("invalid git provider option: %q", gitProvider)
	}

	config := providerConfigs.GetConfig(
		clusterName,
		domainName,
		gitProvider,
		cGitOwner,
		gitProtocol,
		os.Getenv("CF_API_TOKEN"),
		os.Getenv("CF_ORIGIN_CA_ISSUER_API_TOKEN"),
	)

	if _, err := os.Stat(config.SSLBackupDir + "/certificates"); os.IsNotExist(err) {
		// path/to/whatever does not exist
		paths := []string{config.SSLBackupDir + "/certificates", config.SSLBackupDir + "/clusterissuers", config.SSLBackupDir + "/secrets"}

		for _, path := range paths {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				log.Info().Msgf("checking path: %q", path)
				err := os.MkdirAll(path, os.ModePerm)
				if err != nil {
					log.Info().Msg("directory already exists, continuing")
				}
			}
		}
	}

	err := ssl.Backup(config.SSLBackupDir, domainName, config.K1Dir, config.Kubeconfig)
	if err != nil {
		return fmt.Errorf("error backing up SSL resources: %w", err)
	}
	return nil
}
