/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package progress

import (
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/konstructio/kubefirst-api/pkg/types"
	"github.com/konstructio/kubefirst/internal/cluster"
)

// Commands
func GetClusterInterval(clusterName string) tea.Cmd {
	return tea.Every(time.Second*10, func(_ time.Time) tea.Msg {
		provisioningCluster, err := cluster.GetCluster(clusterName)
		if err != nil {
			log.Printf("failed to get cluster %q: %v", clusterName, err)
			return nil
		}

		return CusterProvisioningMsg(provisioningCluster)
	})
}

func AddSuccesMessage(cluster types.Cluster) tea.Cmd {
	return tea.Tick(0, func(_ time.Time) tea.Msg {
		successMessage := DisplaySuccessMessage(cluster)
		printLine(successMessage.message)

		return successMessage
	})
}

func BuildCompletedSteps(cluster types.Cluster) ([]string, string) {
	completedSteps := []string{}
	nextStep := ""
	if cluster.InstallToolsCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.installToolsCheck)
		nextStep = CompletedStepsLabels.domainLivenessCheck
	}
	if cluster.DomainLivenessCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.domainLivenessCheck)
		nextStep = CompletedStepsLabels.kbotSetupCheck
	}
	if cluster.KbotSetupCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.kbotSetupCheck)
		nextStep = CompletedStepsLabels.gitInitCheck
	}
	if cluster.GitInitCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.gitInitCheck)
		nextStep = CompletedStepsLabels.gitopsReadyCheck
	}
	if cluster.GitopsReadyCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.gitopsReadyCheck)
		nextStep = CompletedStepsLabels.gitTerraformApplyCheck
	}
	if cluster.GitTerraformApplyCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.gitTerraformApplyCheck)
		nextStep = CompletedStepsLabels.gitopsPushedCheck
	}
	if cluster.GitopsPushedCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.gitopsPushedCheck)
		nextStep = CompletedStepsLabels.cloudTerraformApplyCheck
	}
	if cluster.CloudTerraformApplyCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.cloudTerraformApplyCheck)
		nextStep = CompletedStepsLabels.clusterSecretsCreatedCheck
	}
	if cluster.ClusterSecretsCreatedCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.clusterSecretsCreatedCheck)
		nextStep = CompletedStepsLabels.argoCDInstallCheck
	}
	if cluster.ArgoCDInstallCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.argoCDInstallCheck)
		nextStep = CompletedStepsLabels.argoCDInitializeCheck
	}
	if cluster.ArgoCDInitializeCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.argoCDInitializeCheck)
		nextStep = CompletedStepsLabels.vaultInitializedCheck
	}
	if cluster.VaultInitializedCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.vaultInitializedCheck)
		nextStep = CompletedStepsLabels.vaultTerraformApplyCheck
	}
	if cluster.VaultTerraformApplyCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.vaultTerraformApplyCheck)
		nextStep = CompletedStepsLabels.usersTerraformApplyCheck
	}
	if cluster.UsersTerraformApplyCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.usersTerraformApplyCheck)
		nextStep = "Wrapping up"
	}

	return completedSteps, nextStep
}
