/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package progress

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	padding  = 2
	maxWidth = 80
)

const debounceDuration = time.Second * 10

var (
	currentPkgNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("211"))
	doneStyle           = lipgloss.NewStyle().Margin(1, 2)
	checkMark           = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
	helpStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render
	StatusStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Render
	spinnerStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
)

var CompletedStepsLabels = ProvisionSteps{
	install_tools_check:           "Installing tools",
	domain_liveness_check:         "Domain liveness check",
	kbot_setup_check:              "Kbot setup",
	git_init_check:                "Initializing Git",
	gitops_ready_check:            "Initializing gitops",
	git_terraform_apply_check:     "Git Terraform apply",
	gitops_pushed_check:           "Gitops repos pushed",
	cloud_terraform_apply_check:   "Cloud Terraform apply",
	cluster_secrets_created_check: "Creating cluster secrets",
	argocd_install_check:          "Installing Argo CD",
	argocd_initialize_check:       "Initializing Argo CD",
	vault_initialized_check:       "Initializing Vault",
	vault_terraform_apply_check:   "Vault Terraform apply",
	users_terraform_apply_check:   "Users Terraform apply",
}
