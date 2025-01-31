/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package provision

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	apiTypes "github.com/konstructio/kubefirst-api/pkg/types"
	utils "github.com/konstructio/kubefirst-api/pkg/utils"
	"github.com/konstructio/kubefirst/internal/cluster"
	"github.com/konstructio/kubefirst/internal/gitShim"
	"github.com/konstructio/kubefirst/internal/launch"
	"github.com/konstructio/kubefirst/internal/step"
	"github.com/konstructio/kubefirst/internal/types"
	"github.com/konstructio/kubefirst/internal/utilities"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func CreateMgmtClusterRequest(gitAuth apiTypes.GitAuth, cliFlags types.CliFlags, catalogApps []apiTypes.GitopsCatalogApp) error {
	clusterRecord, err := utilities.CreateClusterDefinitionRecordFromRaw(
		gitAuth,
		cliFlags,
		catalogApps,
	)

	if err != nil {
		return fmt.Errorf("error creating cluster definition record: %w", err)
	}

	clusterCreated, err := cluster.GetCluster(clusterRecord.ClusterName)
	if err != nil && !errors.Is(err, cluster.ErrNotFound) {
		log.Printf("error retrieving cluster %q: %v", clusterRecord.ClusterName, err)
		return fmt.Errorf("error retrieving cluster: %w", err)
	}

	if errors.Is(err, cluster.ErrNotFound) {
		if err := cluster.CreateCluster(*clusterRecord); err != nil {
			return fmt.Errorf("error creating cluster: %w", err)
		}
	}

	if clusterCreated.Status == "error" {
		cluster.ResetClusterProgress(clusterRecord.ClusterName)
		if err := cluster.CreateCluster(*clusterRecord); err != nil {
			return fmt.Errorf("error re-creating cluster after error state: %w", err)
		}
	}

	return nil
}

type Provisioner struct {
	watcher *ProvisionWatcher
	stepper step.Stepper
}

func NewProvisioner(watcher *ProvisionWatcher, stepper step.Stepper) *Provisioner {
	return &Provisioner{
		watcher: watcher,
		stepper: stepper,
	}
}

func (p *Provisioner) ProvisionManagementCluster(ctx context.Context, cliFlags *types.CliFlags, catalogApps []apiTypes.GitopsCatalogApp) error {

	p.stepper.NewProgressStep("Initialize Configuration")

	clusterSetupComplete := viper.GetBool("kubefirst-checks.cluster-install-complete")
	if clusterSetupComplete {
		p.stepper.InfoStep(step.EmojiCheck, "Cluster already successfully provisioned")
		return nil
	}

	utilities.CreateK1ClusterDirectory(cliFlags.ClusterName)

	p.stepper.NewProgressStep("Validate Git Credentials")

	gitAuth, err := gitShim.ValidateGitCredentials(cliFlags.GitProvider, cliFlags.GithubOrg, cliFlags.GitlabGroup)
	if err != nil {
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
			return fmt.Errorf("failed to initialize Git provider: %w", err)
		}
	}
	viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", cliFlags.GitProvider), true)
	if err = viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write viper config: %w", err)
	}

	p.stepper.NewProgressStep("Setup k3d Cluster")

	k3dClusterCreationComplete := viper.GetBool("launch.deployed")
	isK1Debug := strings.ToLower(os.Getenv("K1_LOCAL_DEBUG")) == "true"

	if !k3dClusterCreationComplete && !isK1Debug {
		if err := launch.Up(ctx, nil, true, cliFlags.UseTelemetry); err != nil {
			return fmt.Errorf("failed to launch k3d cluster: %w", err)
		}
	}

	err = utils.IsAppAvailable(fmt.Sprintf("%s/api/proxyHealth", cluster.GetConsoleIngressURL()), "kubefirst api")
	if err != nil {
		return fmt.Errorf("API availability check failed: %w", err)
	}

	p.stepper.NewProgressStep("Create Management Cluster")

	if err := CreateMgmtClusterRequest(gitAuth, *cliFlags, catalogApps); err != nil {
		return fmt.Errorf("failed to request management cluster creation: %w", err)
	}

	p.stepper.NewProgressStep(p.watcher.GetCurrentStep())

	for !p.watcher.IsComplete() {
		p.stepper.NewProgressStep(p.watcher.GetCurrentStep())
		if err := p.watcher.UpdateProvisionProgress(); err != nil {
			return fmt.Errorf("failed to provision management cluster: %w", err)
		}

		time.Sleep(5 * time.Second)
	}

	return nil
}
