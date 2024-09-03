/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package progress

import (
	"github.com/konstructio/kubefirst-api/pkg/types"
)

// Terminal model
type progressModel struct {
	// Terminal
	error         string
	isProvisioned bool

	header string

	// Provisioning fields
	clusterName         string
	provisioningCluster types.Cluster
	completedSteps      []string
	nextStep            string
	successMessage      string
}

// Bubbletea messages

type ClusterProvisioningMsg types.Cluster

type startProvision struct {
	clusterName string
}

type addStep struct {
	message string
}

type completeStep struct {
	message string
}

type errorMsg struct {
	message string
}

type headerMsg struct {
	message string
}

type successMsg struct {
	message string
}

// Custom

type ProvisionSteps struct {
	installToolsCheck          string
	domainLivenessCheck        string
	kbotSetupCheck             string
	gitInitCheck               string
	gitopsReadyCheck           string
	gitTerraformApplyCheck     string
	gitopsPushedCheck          string
	cloudTerraformApplyCheck   string
	clusterSecretsCreatedCheck string
	argoCDInstallCheck         string
	argoCDInitializeCheck      string
	vaultInitializedCheck      string
	vaultTerraformApplyCheck   string
	usersTerraformApplyCheck   string
}
