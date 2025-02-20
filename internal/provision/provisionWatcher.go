package provision

import (
	"errors"
	"fmt"

	apiTypes "github.com/konstructio/kubefirst-api/pkg/types"
	"github.com/konstructio/kubefirst/internal/cluster"
)

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
	FinalCheck                 = "Final Check"
	ProvisionComplete          = "Provision Complete"
)

type ClusterClient interface {
	GetCluster(clusterName string) (*apiTypes.Cluster, error)
	CreateCluster(cluster apiTypes.ClusterDefinition) error
	ResetClusterProgress(clusterName string) error
}

type Watcher struct {
	clusterName  string
	installSteps []installStep
	client       ClusterClient
}

type installStep struct {
	StepName string
}

func NewProvisionWatcher(clusterName string, client ClusterClient) *Watcher {
	return &Watcher{
		clusterName: clusterName,
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
			{StepName: FinalCheck},
		},
		client: client,
	}
}

func (c *Watcher) GetClusterName() string {
	return c.clusterName
}

func (c *Watcher) SetClusterName(clusterName string) {
	c.clusterName = clusterName
}

func (c *Watcher) IsComplete() bool {
	return len(c.installSteps) == 0
}

func (c *Watcher) GetCurrentStep() string {
	return c.installSteps[0].StepName
}

func (c *Watcher) popStep() string {
	if len(c.installSteps) == 0 {
		return ProvisionComplete
	}

	step := c.installSteps[0]
	c.installSteps = c.installSteps[1:]
	return step.StepName
}

func (c *Watcher) UpdateProvisionProgress() error {
	provisionedCluster, err := c.client.GetCluster(c.clusterName)
	if err != nil {
		if errors.Is(err, cluster.ErrNotFound) {
			return nil
		}

		return fmt.Errorf("error retrieving cluster %q: %w", c.clusterName, err)
	}

	if provisionedCluster.Status == "error" {
		return fmt.Errorf("cluster in error state: %s", provisionedCluster.LastCondition)
	}

	clusterStepStatus := c.mapClusterStepStatus(provisionedCluster)

	if clusterStepStatus[c.GetCurrentStep()] {
		c.popStep()
	}

	return nil
}

func (*Watcher) mapClusterStepStatus(provisionedCluster *apiTypes.Cluster) map[string]bool {
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
		FinalCheck:                 provisionedCluster.FinalCheck,
	}
	return clusterStepStatus
}
