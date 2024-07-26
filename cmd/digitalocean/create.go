/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package digitalocean

import (
	"fmt"
	"os"

	internalssh "github.com/kubefirst/kubefirst-api/pkg/ssh"
	utils "github.com/kubefirst/kubefirst-api/pkg/utils"
	"github.com/kubefirst/kubefirst/internal/catalog"
	"github.com/kubefirst/kubefirst/internal/cluster"
	"github.com/kubefirst/kubefirst/internal/gitShim"
	"github.com/kubefirst/kubefirst/internal/launch"
	"github.com/kubefirst/kubefirst/internal/progress"
	"github.com/kubefirst/kubefirst/internal/provision"
	"github.com/kubefirst/kubefirst/internal/utilities"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func createDigitalocean(cmd *cobra.Command, args []string) error {
	cliFlags, err := utilities.GetFlags(cmd, "digitalocean")
	if err != nil {
		progress.Error(err.Error())
		return nil
	}

	progress.DisplayLogHints(20)

	isValid, catalogApps, err := catalog.ValidateCatalogApps(cliFlags.InstallCatalogApps)
	if !isValid {
		return err
	}

	err = ValidateProvidedFlags(cliFlags.GitProvider)
	if err != nil {
		progress.Error(err.Error())
		return nil
	}

	// If cluster setup is complete, return
	clusterSetupComplete := viper.GetBool("kubefirst-checks.cluster-install-complete")
	if clusterSetupComplete {
		err = fmt.Errorf("this cluster install process has already completed successfully")
		progress.Error(err.Error())
		return nil
	}

	utilities.CreateK1ClusterDirectory(clusterNameFlag)

	gitAuth, err := gitShim.ValidateGitCredentials(cliFlags.GitProvider, cliFlags.GithubOrg, cliFlags.GitlabGroup)

	if err != nil {
		progress.Error(err.Error())
		return nil
	}

	// Validate git
	executionControl := viper.GetBool(fmt.Sprintf("kubefirst-checks.%s-credentials", cliFlags.GitProvider))
	if !executionControl {
		newRepositoryNames := []string{cliFlags.GitopsRepoName, cliFlags.MetaphorRepoName}
		newTeamNames := []string{cliFlags.AdminTeamName, cliFlags.DeveloperTeamName}

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
			return nil
		}
	}
	viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", cliFlags.GitProvider), true)
	viper.WriteConfig()

	k3dClusterCreationComplete := viper.GetBool("launch.deployed")
	if !k3dClusterCreationComplete {
		launch.Up(nil, true, cliFlags.UseTelemetry)
	}

	err = utils.IsAppAvailable(fmt.Sprintf("%s/api/proxyHealth", cluster.GetConsoleIngresUrl()), "kubefirst api")
	if err != nil {
		progress.Error("unable to start kubefirst api")
	}

	provision.CreateMgmtCluster(gitAuth, cliFlags, catalogApps)

	return nil
}

func ValidateProvidedFlags(gitProvider string) error {
	progress.AddStep("Validate provided flags")

	// Validate required environment variables for dns provider
	if dnsProviderFlag == "cloudflare" {
		if os.Getenv("CF_API_TOKEN") == "" {
			return fmt.Errorf("your CF_API_TOKEN environment variable is not set. Please set and try again")
		}
	}

	for _, env := range []string{"DO_TOKEN", "DO_SPACES_KEY", "DO_SPACES_SECRET"} {
		if os.Getenv(env) == "" {
			return fmt.Errorf("your %s variable is unset - please set it before continuing", env)
		}
	}

	switch gitProvider {
	case "github":
		key, err := internalssh.GetHostKey("github.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan github.com >> ~/.ssh/known_hosts` to remedy")
		} else {
			log.Info().Msgf("%s %s\n", "github.com", key.Type())
		}
	case "gitlab":
		key, err := internalssh.GetHostKey("gitlab.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan gitlab.com >> ~/.ssh/known_hosts` to remedy")
		} else {
			log.Info().Msgf("%s %s\n", "gitlab.com", key.Type())
		}
	}

	progress.CompleteStep("Validate provided flags")

	return nil
}
