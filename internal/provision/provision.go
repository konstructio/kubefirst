/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package provision

import (
	"errors"
	"fmt"
	"os"
	"strings"

	apiTypes "github.com/konstructio/kubefirst-api/pkg/types"
	utils "github.com/konstructio/kubefirst-api/pkg/utils"
	"github.com/konstructio/kubefirst/internal/cluster"
	"github.com/konstructio/kubefirst/internal/gitShim"
	"github.com/konstructio/kubefirst/internal/launch"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/konstructio/kubefirst/internal/types"
	"github.com/konstructio/kubefirst/internal/utilities"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func CreateMgmtClusterRequest(gitAuth apiTypes.GitAuth, cliFlags types.CliFlags, catalogApps []apiTypes.GitopsCatalogApp) error {
	clusterRecord := utilities.CreateClusterDefinitionRecordFromRaw(
		gitAuth,
		cliFlags,
		catalogApps,
	)

	clusterCreated, err := cluster.GetCluster(clusterRecord.ClusterName)
	if err != nil && !errors.Is(err, cluster.ErrNotFound) {
		log.Printf("error retrieving cluster %q: %v", clusterRecord.ClusterName, err)
		return fmt.Errorf("error retrieving cluster: %w", err)
	}

	if errors.Is(err, cluster.ErrNotFound) {
		if err := cluster.CreateCluster(clusterRecord); err != nil {
			progress.Error(err.Error())
			return fmt.Errorf("error creating cluster: %w", err)
		}
	}

	if clusterCreated.Status == "error" {
		cluster.ResetClusterProgress(clusterRecord.ClusterName)
		if err := cluster.CreateCluster(clusterRecord); err != nil {
			progress.Error(err.Error())
			return fmt.Errorf("error re-creating cluster after error state: %w", err)
		}
	}

	progress.StartProvisioning(clusterRecord.ClusterName)
	return nil
}

func ManagementCluster(cliFlags types.CliFlags, catalogApps []apiTypes.GitopsCatalogApp) error {
	clusterSetupComplete := viper.GetBool("kubefirst-checks.cluster-install-complete")
	if clusterSetupComplete {
		err := fmt.Errorf("this cluster install process has already completed successfully")
		progress.Error(err.Error())
		return nil
	}

	utilities.CreateK1ClusterDirectory(cliFlags.ClusterName)

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
			return fmt.Errorf("failed to initialize Git provider: %w", err)
		}
	}
	viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", cliFlags.GitProvider), true)
	if err = viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write viper config: %w", err)
	}

	k3dClusterCreationComplete := viper.GetBool("launch.deployed")
	isK1Debug := strings.ToLower(os.Getenv("K1_LOCAL_DEBUG")) == "true"

	if !k3dClusterCreationComplete && !isK1Debug {
		launch.Up(nil, true, cliFlags.UseTelemetry)
	}

	err = utils.IsAppAvailable(fmt.Sprintf("%s/api/proxyHealth", cluster.GetConsoleIngressURL()), "kubefirst api")
	if err != nil {
		progress.Error("unable to start kubefirst api")
		return fmt.Errorf("API availability check failed: %w", err)
	}

	if err := CreateMgmtClusterRequest(gitAuth, cliFlags, catalogApps); err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to create management cluster: %w", err)
	}

	return nil
}
