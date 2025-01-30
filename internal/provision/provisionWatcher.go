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
	ProvisionComplete          = "Provision Complete"
)

type ClusterClient interface {
	GetCluster(clusterName string) (*apiTypes.Cluster, error)
	CreateCluster(cluster apiTypes.ClusterDefinition) error
	ResetClusterProgress(clusterName string) error
}

type ProvisionWatcher struct {
	clusterName  string
	installSteps []installStep
	client       ClusterClient
}

type installStep struct {
	StepName string
}

func NewProvisionWatcher(clusterName string, client ClusterClient) *ProvisionWatcher {
	return &ProvisionWatcher{clusterName: clusterName,
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

func (c *ProvisionWatcher) GetClusterName() string {
	return c.clusterName
}

func (c *ProvisionWatcher) SetClusterName(clusterName string) {
	c.clusterName = clusterName
}

func (c *ProvisionWatcher) IsComplete() bool {
	return len(c.installSteps) == 0
}

func (c *ProvisionWatcher) GetCurrentStep() string {
	return c.installSteps[0].StepName
}

func (c *ProvisionWatcher) popStep() string {
	if len(c.installSteps) == 0 {
		return ProvisionComplete
	}

	step := c.installSteps[0]
	c.installSteps = c.installSteps[1:]
	return step.StepName
}

func (c *ProvisionWatcher) UpdateProvisionProgress() error {
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

func (*ProvisionWatcher) mapClusterStepStatus(provisionedCluster *apiTypes.Cluster) map[string]bool {
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
	return clusterStepStatus
}
