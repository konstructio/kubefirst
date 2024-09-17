/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package akamai

import (
	"fmt"
	"os"
	"strings"

	internalssh "github.com/konstructio/kubefirst-api/pkg/ssh"
	pkg "github.com/konstructio/kubefirst-api/pkg/utils"
	"github.com/konstructio/kubefirst/internal/catalog"
	"github.com/konstructio/kubefirst/internal/cluster"
	"github.com/konstructio/kubefirst/internal/docker"
	"github.com/konstructio/kubefirst/internal/gitShim"
	"github.com/konstructio/kubefirst/internal/launch"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/konstructio/kubefirst/internal/provision"
	"github.com/konstructio/kubefirst/internal/utilities"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func createAkamai(cmd *cobra.Command, args []string) error {
	cliFlags, err := utilities.GetFlags(cmd, "akamai")
	if err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to get flags: %w", err)
	}

	log.Info().Msg("Check Docker status")
	err = docker.Checkstatus()
	if err != nil {
		log.Info().Msgf("%s", err)
		return err
	}

	progress.DisplayLogHints(25)

	isValid, catalogApps, err := catalog.ValidateCatalogApps(cliFlags.InstallCatalogApps)
	if !isValid {
		return fmt.Errorf("catalog validation failed: %w", err)
	}

	err = ValidateProvidedFlags(cliFlags.GitProvider)
	if err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to validate provided flags: %w", err)
	}

	// If cluster setup is complete, return

	utilities.CreateK1ClusterDirectory(clusterNameFlag)

	gitAuth, err := gitShim.ValidateGitCredentials(cliFlags.GitProvider, cliFlags.GithubOrg, cliFlags.GitlabGroup)
	if err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to validate git credentials: %w", err)
	}

	// Validate git
	executionControl := viper.GetBool(fmt.Sprintf("kubefirst-checks.%s-credentials", cliFlags.GitProvider))
	if !executionControl {
		newRepositoryNames := []string{"gitops", "metaphor"}
		newTeamNames := []string{"admins", "developers"}

		initGitParameters := gitShim.GitInitParameters{
			GitProvider:  cliFlags.GitProvider,
			GitToken:     gitAuth.Token,
			GitOwner:     gitAuth.Owner,
			Repositories: newRepositoryNames,
			Teams:        newTeamNames,
		}

		err = gitShim.InitializeGitProvider(&initGitParameters)
		if err != nil {
			progress.Error(err.Error())
			return fmt.Errorf("failed to initialize git provider: %w", err)
		}
	}
	viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", cliFlags.GitProvider), true)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write viper config: %w", err)
	}

	k3dClusterCreationComplete := viper.GetBool("launch.deployed")
	isK1Debug := strings.ToLower(os.Getenv("K1_LOCAL_DEBUG")) == "true"

	if !k3dClusterCreationComplete && !isK1Debug {
		launch.Up(nil, true, cliFlags.UseTelemetry)
	}

	err = pkg.IsAppAvailable(fmt.Sprintf("%s/api/proxyHealth", cluster.GetConsoleIngresUrl()), "kubefirst api")
	if err != nil {
		progress.Error("unable to start kubefirst api")
		return fmt.Errorf("failed to check kubefirst api availability: %w", err)
	}

	provision.CreateMgmtCluster(gitAuth, cliFlags, catalogApps)
	return nil
}

func ValidateProvidedFlags(gitProvider string) error {
	progress.AddStep("Validate provided flags")

	if os.Getenv("LINODE_TOKEN") == "" {
		return fmt.Errorf("your LINODE_TOKEN is not set - please set and re-run your last command")
	}

	// Validate required environment variables for dns provider
	if dnsProviderFlag == "cloudflare" {
		if os.Getenv("CF_API_TOKEN") == "" {
			return fmt.Errorf("your CF_API_TOKEN environment variable is not set. Please set and try again")
		}
	}

	switch gitProvider {
	case "github":
		key, err := internalssh.GetHostKey("github.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan github.com >> ~/.ssh/known_hosts` to remedy: %w", err)
		} else {
			log.Info().Msgf("%q %s", "github.com", key.Type())
		}
	case "gitlab":
		key, err := internalssh.GetHostKey("gitlab.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan gitlab.com >> ~/.ssh/known_hosts` to remedy: %w", err)
		} else {
			log.Info().Msgf("%q %s", "gitlab.com", key.Type())
		}
	}

	progress.CompleteStep("Validate provided flags")

	return nil
}
