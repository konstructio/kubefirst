/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package progress

var CompletedStepsLabels = ProvisionSteps{
	installToolsCheck:          "Installing tools",
	domainLivenessCheck:        "Domain liveness check",
	kbotSetupCheck:             "Kbot setup",
	gitInitCheck:               "Initializing Git",
	gitopsReadyCheck:           "Initializing GitOps",
	gitTerraformApplyCheck:     "Git Terraform apply",
	gitopsPushedCheck:          "GitOps repos pushed",
	cloudTerraformApplyCheck:   "Cloud Terraform apply",
	clusterSecretsCreatedCheck: "Creating cluster secrets",
	argoCDInstallCheck:         "Installing ArgoCD",
	argoCDInitializeCheck:      "Initializing ArgoCD",
	vaultInitializedCheck:      "Initializing Vault",
	vaultTerraformApplyCheck:   "Vault Terraform apply",
	usersTerraformApplyCheck:   "Users Terraform apply",
}
