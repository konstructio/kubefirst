/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package progress

type ProvisionSteps struct {
	install_tools_check           string
	domain_liveness_check         string
	kbot_setup_check              string
	git_init_check                string
	gitops_ready_check            string
	git_terraform_apply_check     string
	gitops_pushed_check           string
	cloud_terraform_apply_check   string
	cluster_secrets_created_check string
	argocd_install_check          string
	argocd_initialize_check       string
	vault_initialized_check       string
	vault_terraform_apply_check   string
	users_terraform_apply_check   string
}
