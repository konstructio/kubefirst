/*
Copyright (C) 2021-2024, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package azure

import (
	"fmt"
	"os"
	"strings"

	internalssh "github.com/konstructio/kubefirst-api/pkg/ssh"
	utils "github.com/konstructio/kubefirst-api/pkg/utils"
	"github.com/konstructio/kubefirst/internal/catalog"
	"github.com/konstructio/kubefirst/internal/cluster"
	"github.com/konstructio/kubefirst/internal/gitShim"
	"github.com/konstructio/kubefirst/internal/launch"
	"github.com/konstructio/kubefirst/internal/provision"
	"github.com/konstructio/kubefirst/internal/utilities"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Environment variables required for authentication. This should be a
// service principal - the Terraform provider docs detail how to create
// one
// @link https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/guides/service_principal_client_secret.html
var envvarSecrets = []string{
	"ARM_CLIENT_ID",
	"ARM_CLIENT_SECRET",
	"ARM_TENANT_ID",
	"ARM_SUBSCRIPTION_ID",
}

func createAzure(cmd *cobra.Command, _ []string) error {
	cliFlags, err := utilities.GetFlags(cmd, "azure")
	if err != nil {
		return fmt.Errorf("failed to get flags: %w", err)
	}

	// TODO: Handle for non-bubbletea
	// progress.DisplayLogHints(20)

	isValid, catalogApps, err := catalog.ValidateCatalogApps(cliFlags.InstallCatalogApps)
	if !isValid {
		return fmt.Errorf("failed to validate catalog apps")
	}

	err = ValidateProvidedFlags(cliFlags.GitProvider)
	if err != nil {
		return fmt.Errorf("failed to validate flags: %w", err)
	}

	// If cluster setup is complete, return
	clusterSetupComplete := viper.GetBool("kubefirst-checks.cluster-install-complete")
	if clusterSetupComplete {
		err = fmt.Errorf("this cluster install process has already completed successfully")
		return err
	}

	utilities.CreateK1ClusterDirectory(cliFlags.ClusterName)

	gitAuth, err := gitShim.ValidateGitCredentials(cliFlags.GitProvider, cliFlags.GithubOrg, cliFlags.GitlabGroup)
	if err != nil {
		return fmt.Errorf("failed to validate git credentials: %w", err)
	}

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
			return fmt.Errorf("failed to initialize git provider: %w", err)
		}
	}

	viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", cliFlags.GitProvider), true)
	viper.WriteConfig()

	k3dClusterCreationComplete := viper.GetBool("launch.deployed")
	isK1Debug := strings.ToLower(os.Getenv("K1_LOCAL_DEBUG")) == "true"

	if !k3dClusterCreationComplete && !isK1Debug {
		launch.Up(nil, true, cliFlags.UseTelemetry)
	}

	err = utils.IsAppAvailable(fmt.Sprintf("%s/api/proxyHealth", cluster.GetConsoleIngressURL()), "kubefirst api")
	if err != nil {
		return fmt.Errorf("unable to start kubefirst api, error: %w", err)
	}

	provision.CreateMgmtCluster(gitAuth, cliFlags, catalogApps)

	return nil
}

func ValidateProvidedFlags(gitProvider string) error {

	// TODO: Handle for non-bubbletea
	// progress.AddStep("Validate provided flags")

	for _, env := range envvarSecrets {
		if os.Getenv(env) == "" {
			return fmt.Errorf("your %s is not set - please set and re-run your last command", env)
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

	// TODO: Handle for non-bubbletea
	// progress.CompleteStep("Validate provided flags")

	return nil
}
