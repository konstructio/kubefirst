/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package progress

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/kubefirst/internal/cluster"
)

// Commands
func GetClusterInterval(clusterName string) tea.Cmd {
	return tea.Every(time.Second*10, func(t time.Time) tea.Msg {
		provisioningCluster, err := cluster.GetCluster(clusterName)

		if err != nil {

		}

		return CusterProvisioningMsg(provisioningCluster)
	})
}

func AddSuccesMessage(cluster types.Cluster) tea.Cmd {
	return tea.Tick(0, func(t time.Time) tea.Msg {
		successMessage := DisplaySuccessMessage(cluster)

		return successMsg(successMessage)
	})
}

func BuildCompletedSteps(cluster types.Cluster, model progressModel) ([]string, string) {
	completedSteps := []string{}
	nextStep := ""
	if cluster.InstallToolsCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.install_tools_check)
		nextStep = CompletedStepsLabels.domain_liveness_check
	}
	if cluster.DomainLivenessCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.domain_liveness_check)
		nextStep = CompletedStepsLabels.kbot_setup_check
	}
	if cluster.KbotSetupCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.kbot_setup_check)
		nextStep = CompletedStepsLabels.git_init_check
	}
	if cluster.GitInitCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.git_init_check)
		nextStep = CompletedStepsLabels.gitops_ready_check
	}
	if cluster.GitopsReadyCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.gitops_ready_check)
		nextStep = CompletedStepsLabels.git_terraform_apply_check
	}
	if cluster.GitTerraformApplyCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.git_terraform_apply_check)
		nextStep = CompletedStepsLabels.gitops_pushed_check
	}
	if cluster.GitopsPushedCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.gitops_pushed_check)
		nextStep = CompletedStepsLabels.cloud_terraform_apply_check
	}
	if cluster.CloudTerraformApplyCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.cloud_terraform_apply_check)
		nextStep = CompletedStepsLabels.cluster_secrets_created_check
	}
	if cluster.ClusterSecretsCreatedCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.cluster_secrets_created_check)
		nextStep = CompletedStepsLabels.argocd_install_check
	}
	if cluster.ArgoCDInstallCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.argocd_install_check)
		nextStep = CompletedStepsLabels.argocd_initialize_check
	}
	if cluster.ArgoCDInitializeCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.argocd_initialize_check)
		nextStep = CompletedStepsLabels.vault_initialized_check
	}
	if cluster.VaultInitializedCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.vault_initialized_check)
		nextStep = CompletedStepsLabels.vault_terraform_apply_check
	}
	if cluster.VaultTerraformApplyCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.vault_terraform_apply_check)
		nextStep = CompletedStepsLabels.users_terraform_apply_check
	}
	if cluster.UsersTerraformApplyCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.users_terraform_apply_check)
		nextStep = "Wrapping up"
	}

	return completedSteps, nextStep
}
