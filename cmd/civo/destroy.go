/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"fmt"
	"os"
	"strings"

	"github.com/kubefirst/kubefirst/internal/launch"
	"github.com/kubefirst/kubefirst/internal/progress"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/providerConfigs"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func destroyCivo(cmd *cobra.Command, args []string) error {
	progress.DisplayLogHints()

	// Determine if there are active installs
	gitProvider := viper.GetString("flags.git-provider")
	gitProtocol := viper.GetString("flags.git-protocol")

	log.Info().Msg("destroying kubefirst platform in civo")

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

	// Instantiate civo config
	config := providerConfigs.GetConfig(
		clusterName,
		domainName,
		gitProvider,
		cGitOwner,
		gitProtocol,
		os.Getenv("CF_API_TOKEN"),
		os.Getenv("CF_ORIGIN_CA_ISSUER_API_TOKEN"),
	)
	config.CivoToken = os.Getenv("CIVO_TOKEN")
	switch gitProvider {
	case "github":
		config.GithubToken = cGitToken
	case "gitlab":
		config.GitlabToken = cGitToken
	}

	if len(cGitToken) == 0 {
		return fmt.Errorf(
			"please set a %s_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/%s/install.html#step-3-kubefirst-init",
			strings.ToUpper(gitProvider), gitProvider,
		)
	}
	if len(config.CivoToken) == 0 {
		return fmt.Errorf("\n\nYour CIVO_TOKEN environment variable isn't set,\nvisit this link https://dashboard.civo.com/security and set the environment variable")
	}

	launch.Down(true)

	err := pkg.ResetK1Dir(config.K1Dir)
	if err != nil {
		return err
	}
	log.Info().Msg("previous platform content removed")

	log.Info().Msg("resetting `$HOME/.kubefirst` config")
	viper.Set("argocd", "")
	viper.Set(gitProvider, "")
	viper.Set("components", "")
	viper.Set("kbot", "")
	viper.Set("kubefirst-checks", "")
	viper.Set("kubefirst", "")
	viper.WriteConfig()

	if _, err := os.Stat(config.K1Dir + "/kubeconfig"); !os.IsNotExist(err) {
		err = os.Remove(config.K1Dir + "/kubeconfig")
		if err != nil {
			return fmt.Errorf("unable to delete %q folder, error: %s", config.K1Dir+"/kubeconfig", err)
		}
	}

	progress.Success("Your kubefirst platform has been destroyed.")

	return nil
}
