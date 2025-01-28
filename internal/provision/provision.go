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
	"github.com/konstructio/kubefirst/internal/cluster"
	"github.com/konstructio/kubefirst/internal/gitShim"
	"github.com/konstructio/kubefirst/internal/launch"
	"github.com/konstructio/kubefirst/internal/types"
	"github.com/konstructio/kubefirst/internal/utilities"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func CreateMgmtCluster(gitAuth apiTypes.GitAuth, cliFlags types.CliFlags, catalogApps []apiTypes.GitopsCatalogApp) error {
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
		return fmt.Errorf("error retrieving cluster %q: %w", clusterRecord.ClusterName, err)
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

const (
	InstallToolsCheck          = "Install Tools"
	DomainLivenessCheck        = "Domain Liveness"
	KBotSetupCheck             = "KBot Setup"
	GitInitCheck               = "Git Init"
	GitOpsReadyCheck           = "GitOps Ready"
	GitTerraformApplyCheck     = "Git Terraform Apply"
	GitOpsPushedCheck          = "GitOps Pushed"
	CloudTerraformApplyCheck   = "Cloud Terraform Apply"
	ClusterSecretsCreatedCheck = "Cluster Secrets Created"
	ArgoCDInstallCheck         = "ArgoCD Install"
	ArgoCDInitializeCheck      = "ArgoCD Initialize"
	VaultInitializedCheck      = "Vault Initialized"
	VaultTerraformApplyCheck   = "Vault Terraform Apply"
	UsersTerraformApplyCheck   = "Users Terraform Apply"
	ProvisionComplete          = "Provision Complete"
)

type ClusterClient interface {
	GetCluster(clusterName string) (apiTypes.Cluster, error)
	CreateCluster(cluster apiTypes.ClusterDefinition) error
	ResetClusterProgress(clusterName string) error
}

type ClusterProvision struct {
	clusterName  string
	installSteps []installStep
	client       ClusterClient
}

type installStep struct {
	StepName string
}

func NewClusterProvision(clusterName string, client ClusterClient) *ClusterProvision {
	return &ClusterProvision{clusterName: clusterName,
		installSteps: []installStep{
			{StepName: InstallToolsCheck},
			{StepName: DomainLivenessCheck},
			{StepName: KBotSetupCheck},
			{StepName: GitInitCheck},
			{StepName: GitOpsReadyCheck},
			{StepName: GitTerraformApplyCheck},
			{StepName: GitOpsPushedCheck},
			{StepName: CloudTerraformApplyCheck},
			{StepName: ClusterSecretsCreatedCheck},
			{StepName: ArgoCDInstallCheck},
			{StepName: ArgoCDInitializeCheck},
			{StepName: VaultInitializedCheck},
			{StepName: VaultTerraformApplyCheck},
			{StepName: UsersTerraformApplyCheck},
		},
		client: client,
	}
}

func (c *ClusterProvision) GetInstallSteps() []installStep {
	return c.installSteps
}

func (c *ClusterProvision) IsComplete() bool {
	return len(c.installSteps) == 0
}

func (c *ClusterProvision) GetCurrentStep() string {
	return c.installSteps[0].StepName
}

func (c *ClusterProvision) PopStep() string {
	if len(c.installSteps) == 0 {
		return ProvisionComplete
	}

	step := c.installSteps[0]
	c.installSteps = c.installSteps[1:]
	return step.StepName
}

func (c *ClusterProvision) UpdateProvisionProgress() error {
	provisionedCluster, err := c.client.GetCluster(c.clusterName)
	if err != nil {
		if errors.Is(err, cluster.ErrNotFound) {
			return nil
		}

		log.Printf("error retrieving cluster %q: %v", provisionedCluster.ClusterName, err)
		return fmt.Errorf("error retrieving cluster %q: %w", provisionedCluster.ClusterName, err)
	}

	if provisionedCluster.Status == "error" {
		return fmt.Errorf("cluster in error state: %s", provisionedCluster.LastCondition)
	}

	clusterStepStatus := map[string]bool{
		InstallToolsCheck:          provisionedCluster.InstallToolsCheck,
		DomainLivenessCheck:        provisionedCluster.DomainLivenessCheck,
		KBotSetupCheck:             provisionedCluster.KbotSetupCheck,
		GitInitCheck:               provisionedCluster.GitInitCheck,
		GitOpsReadyCheck:           provisionedCluster.GitopsReadyCheck,
		GitTerraformApplyCheck:     provisionedCluster.GitTerraformApplyCheck,
		GitOpsPushedCheck:          provisionedCluster.GitopsPushedCheck,
		CloudTerraformApplyCheck:   provisionedCluster.CloudTerraformApplyCheck,
		ClusterSecretsCreatedCheck: provisionedCluster.ClusterSecretsCreatedCheck,
		ArgoCDInstallCheck:         provisionedCluster.ArgoCDInstallCheck,
		ArgoCDInitializeCheck:      provisionedCluster.ArgoCDInitializeCheck,
		VaultInitializedCheck:      provisionedCluster.VaultInitializedCheck,
		VaultTerraformApplyCheck:   provisionedCluster.VaultTerraformApplyCheck,
		UsersTerraformApplyCheck:   provisionedCluster.UsersTerraformApplyCheck,
	}

	if clusterStepStatus[c.GetCurrentStep()] {
		c.PopStep()
	}

	return nil
}

type KubefirstClient interface {
	CreateManagementCluster(ctx context.Context, catalogApps []apiTypes.GitopsCatalogApp) error
}

func CreateManagementCluster(c KubefirstCivoClient, catalogApps []apiTypes.GitopsCatalogApp) error {

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

	if err := CreateMgmtCluster(gitAuth, c.cliFlags, catalogApps); err != nil {
		wrerr := fmt.Errorf("failed to create management cluster: %w", err)
		createMgmtClusterStep.Complete(wrerr)
		return wrerr
	}

	createMgmtClusterStep.Complete(nil)

	clusterClient := cluster.ClusterClient{}

	clusterProvision := NewClusterProvision(c.cliFlags.ClusterName, &clusterClient)

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
