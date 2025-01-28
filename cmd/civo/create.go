/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	internalssh "github.com/konstructio/kubefirst-api/pkg/ssh"
	apiTypes "github.com/konstructio/kubefirst-api/pkg/types"
	utils "github.com/konstructio/kubefirst-api/pkg/utils"
	"github.com/konstructio/kubefirst/internal/cluster"
	"github.com/konstructio/kubefirst/internal/gitShim"
	"github.com/konstructio/kubefirst/internal/launch"
	"github.com/konstructio/kubefirst/internal/provision"
	"github.com/konstructio/kubefirst/internal/step"
	"github.com/konstructio/kubefirst/internal/types"
	"github.com/konstructio/kubefirst/internal/utilities"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type KubefirstCivoClient struct {
	stepper  step.Stepper
	cliFlags types.CliFlags
}

func (kc *KubefirstCivoClient) CreateManagementCluster(ctx context.Context, catalogApps []apiTypes.GitopsCatalogApp) error {

	err := ValidateProvidedFlags(kc.cliFlags.GitProvider, kc.cliFlags.DNSProvider)

	if err != nil {
		return fmt.Errorf("failed to validate provided flags: %w", err)
	}

	return CreateManagementCluster(kc, catalogApps)
}

func CreateManagementCluster(c *KubefirstCivoClient, catalogApps []apiTypes.GitopsCatalogApp) error {

	initializeConfigStep := c.stepper.NewProgressStep("Initialize Config")

	utilities.CreateK1ClusterDirectory(c.cliFlags.ClusterName)

	gitAuth, err := gitShim.ValidateGitCredentials(c.cliFlags.GitProvider, c.cliFlags.GithubOrg, c.cliFlags.GitlabGroup)
	if err != nil {
		wrerr := fmt.Errorf("failed to validate git credentials: %w", err)
		initializeConfigStep.Complete(wrerr)
		return wrerr
	}

	initializeConfigStep.Complete(nil)
	validateGitStep := c.stepper.NewProgressStep("Setup Gitops Repository")

	// Validate git
	executionControl := viper.GetBool(fmt.Sprintf("kubefirst-checks.%s-credentials", c.cliFlags.GitProvider))

	if !executionControl {
		newRepositoryNames := []string{"gitops", "metaphor"}
		newTeamNames := []string{"admins", "developers"}

		initGitParameters := gitShim.GitInitParameters{
			GitProvider:  c.cliFlags.GitProvider,
			GitToken:     gitAuth.Token,
			GitOwner:     gitAuth.Owner,
			Repositories: newRepositoryNames,
			Teams:        newTeamNames,
		}

		err = gitShim.InitializeGitProvider(&initGitParameters)
		if err != nil {
			wrerr := fmt.Errorf("failed to initialize Git provider: %w", err)
			validateGitStep.Complete(wrerr)
			return wrerr
		}
	}

	viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", c.cliFlags.GitProvider), true)

	viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", c.cliFlags.GitProvider), true)

	if err = viper.WriteConfig(); err != nil {
		wrerr := fmt.Errorf("failed to write viper config: %w", err)
		validateGitStep.Complete(wrerr)
		return wrerr
	}

	validateGitStep.Complete(nil)
	setupK3dClusterStep := c.stepper.NewProgressStep("Setup k3d Cluster")

	k3dClusterCreationComplete := viper.GetBool("launch.deployed")
	isK1Debug := strings.ToLower(os.Getenv("K1_LOCAL_DEBUG")) == "true"

	if !k3dClusterCreationComplete && !isK1Debug {
		err = launch.Up(nil, true, c.cliFlags.UseTelemetry)

		if err != nil {
			wrerr := fmt.Errorf("failed to setup k3d cluster: %w", err)
			setupK3dClusterStep.Complete(wrerr)
			return wrerr
		}
	}

	err = utils.IsAppAvailable(fmt.Sprintf("%s/api/proxyHealth", cluster.GetConsoleIngressURL()), "kubefirst api")
	if err != nil {
		wrerr := fmt.Errorf("API availability check failed: %w", err)
		setupK3dClusterStep.Complete(wrerr)
		return wrerr
	}

	setupK3dClusterStep.Complete(nil)
	createMgmtClusterStep := c.stepper.NewProgressStep("Create Management Cluster")

	if err := provision.CreateMgmtCluster(gitAuth, c.cliFlags, catalogApps); err != nil {
		wrerr := fmt.Errorf("failed to create management cluster: %w", err)
		createMgmtClusterStep.Complete(wrerr)
		return wrerr
	}

	createMgmtClusterStep.Complete(nil)

	clusterClient := cluster.ClusterClient{}

	clusterProvision := provision.NewClusterProvision(c.cliFlags.ClusterName, &clusterClient)

	currentClusterStep := c.stepper.NewProgressStep(clusterProvision.GetCurrentStep())

	for !clusterProvision.IsComplete() {

		if currentClusterStep.GetName() != clusterProvision.GetCurrentStep() {
			currentClusterStep.Complete(nil)
			currentClusterStep = c.stepper.NewProgressStep(clusterProvision.GetCurrentStep())
		}

		err = clusterProvision.UpdateProvisionProgress()

		if err != nil {
			wrerr := fmt.Errorf("failure provisioning the management cluster: %w", err)
			currentClusterStep.Complete(wrerr)
			return wrerr
		}

		time.Sleep(5 * time.Second)
	}

	return nil
}

func ValidateProvidedFlags(gitProvider, dnsProvider string) error {

	if os.Getenv("CIVO_TOKEN") == "" {
		return fmt.Errorf("your CIVO_TOKEN is not set - please set and re-run your last command")
	}

	// Validate required environment variables for dns provider
	if dnsProvider == "cloudflare" {
		if os.Getenv("CF_API_TOKEN") == "" {
			return fmt.Errorf("your CF_API_TOKEN environment variable is not set. Please set and try again")
		}
	}

	switch gitProvider {
	case "github":
		key, err := internalssh.GetHostKey("github.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan github.com >> ~/.ssh/known_hosts` to remedy")
		}
		log.Info().Msgf("github.com %q", key.Type())
	case "gitlab":
		key, err := internalssh.GetHostKey("gitlab.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan gitlab.com >> ~/.ssh/known_hosts` to remedy")
		}
		log.Info().Msgf("gitlab.com %q", key.Type())
	}

	return nil
}
