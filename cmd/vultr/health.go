/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vultr

import (
	"context"
	"os"

	"github.com/kubefirst/runtime/pkg/vultr"
	"github.com/kubefirst/runtime/pkg/providerConfigs"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// checkVultrCloudHealth returns relevant info regarding Vultr prior to executing
// certain commands
func checkVultrCloudHealth(cmd *cobra.Command, args []string) error {
	cloudRegion, err := cmd.Flags().GetString("cloud-region")
	if err != nil {
		return err
	}

	// Instantiate vultr config
	gitProvider := viper.GetString("flags.git-provider")
	clusterName := viper.GetString("flags.cluster-name")
	domainName := viper.GetString("flags.domain-name")

	// Switch based on git provider, set params
	var cGitOwner, cGitToken string
	switch gitProvider {
	case "github":
		cGitOwner = viper.GetString("flags.github-owner")
		cGitToken = os.Getenv("GITHUB_TOKEN")
	case "gitlab":
		cGitOwner = viper.GetString("flags.gitlab-owner")
		cGitToken = os.Getenv("GITLAB_TOKEN")
	default:
		log.Panic().Msgf("invalid git provider option")
	}

	config := providerConfigs.GetConfig(clusterName, domainName, gitProvider, cGitOwner, gitProtocolFlag)
	config.VultrToken = os.Getenv("VULTR_API_KEY")
	switch gitProvider {
	case "github":
		config.GithubToken = cGitToken
	case "gitlab":
		config.GitlabToken = cGitToken
	}

	vultrConf := vultr.VultrConfiguration{
		Client:  vultr.NewVultr(config.VultrToken),
		Context: context.Background()
	}
	err = vultrConf.HealthCheck(cloudRegion)
	if err != nil {
		return err
	}

	return nil
}
