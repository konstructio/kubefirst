/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vultr

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/vault"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
)

func readVaultTokenFromSecret(clientset *kubernetes.Clientset, config *VultrConfig) string {
	existingKubernetesSecret, err := k8s.ReadSecretV2(clientset, vault.VaultNamespace, vault.VaultSecretName)
	if err != nil || existingKubernetesSecret == nil {
		log.Printf("Error reading existing Secret data: %s", err)
		return ""
	}

	return existingKubernetesSecret["root-token"]
}

func GetVultrTerraformEnvs(envs map[string]string) map[string]string {
	envs["VULTR_API_KEY"] = os.Getenv("VULTR_API_KEY")
	envs["AWS_ACCESS_KEY_ID"] = viper.GetString("kubefirst.state-store-creds.access-key-id")
	envs["AWS_SECRET_ACCESS_KEY"] = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")
	envs["TF_VAR_aws_access_key_id"] = viper.GetString("kubefirst.state-store-creds.access-key-id")
	envs["TF_VAR_aws_secret_access_key"] = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")
	//envs["TF_LOG"] = "debug"

	return envs
}

func GetGithubTerraformEnvs(envs map[string]string) map[string]string {
	envs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
	envs["GITHUB_OWNER"] = viper.GetString("flags.github-owner")
	envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
	envs["TF_VAR_kbot_ssh_public_key"] = viper.GetString("kbot.public-key")
	envs["VULTR_API_KEY"] = os.Getenv("VULTR_API_KEY")
	envs["AWS_ACCESS_KEY_ID"] = viper.GetString("kubefirst.state-store-creds.access-key-id")
	envs["AWS_SECRET_ACCESS_KEY"] = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")
	envs["TF_VAR_aws_access_key_id"] = viper.GetString("kubefirst.state-store-creds.access-key-id")
	envs["TF_VAR_aws_secret_access_key"] = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")

	return envs
}

func GetGitlabTerraformEnvs(envs map[string]string, gid int) map[string]string {
	envs["GITLAB_TOKEN"] = os.Getenv("GITLAB_TOKEN")
	envs["GITLAB_OWNER"] = viper.GetString("flags.gitlab-owner")
	envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
	envs["TF_VAR_atlantis_repo_webhook_url"] = viper.GetString("gitlab.atlantis.webhook.url")
	envs["TF_VAR_kbot_ssh_public_key"] = viper.GetString("kbot.public-key")
	envs["VULTR_API_KEY"] = os.Getenv("VULTR_API_KEY")
	envs["AWS_ACCESS_KEY_ID"] = viper.GetString("kubefirst.state-store-creds.access-key-id")
	envs["AWS_SECRET_ACCESS_KEY"] = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")
	envs["TF_VAR_aws_access_key_id"] = viper.GetString("kubefirst.state-store-creds.access-key-id")
	envs["TF_VAR_aws_secret_access_key"] = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")
	envs["TF_VAR_owner_group_id"] = strconv.Itoa(gid)
	envs["TF_VAR_gitlab_owner"] = viper.GetString("flags.gitlab-owner")

	return envs
}

func GetUsersTerraformEnvs(clientset *kubernetes.Clientset, config *VultrConfig, envs map[string]string) map[string]string {
	var tokenValue string
	switch config.GitProvider {
	case "github":
		tokenValue = config.GithubToken
	case "gitlab":
		tokenValue = config.GitlabToken
	}
	envs["VAULT_TOKEN"] = readVaultTokenFromSecret(clientset, config)
	envs["VAULT_ADDR"] = VaultPortForwardURL
	envs[fmt.Sprintf("%s_TOKEN", strings.ToUpper(config.GitProvider))] = tokenValue
	envs[fmt.Sprintf("%s_OWNER", strings.ToUpper(config.GitProvider))] = viper.GetString(fmt.Sprintf("flags.%s-owner", config.GitProvider))
	envs["VULTR_API_KEY"] = os.Getenv("VULTR_API_KEY")
	envs["AWS_ACCESS_KEY_ID"] = viper.GetString("kubefirst.state-store-creds.access-key-id")
	envs["AWS_SECRET_ACCESS_KEY"] = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")
	envs["TF_VAR_aws_access_key_id"] = viper.GetString("kubefirst.state-store-creds.access-key-id")
	envs["TF_VAR_aws_secret_access_key"] = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")

	return envs
}

func GetVaultTerraformEnvs(clientset *kubernetes.Clientset, config *VultrConfig, envs map[string]string) map[string]string {
	var tokenValue string
	switch config.GitProvider {
	case "github":
		tokenValue = config.GithubToken
	case "gitlab":
		tokenValue = config.GitlabToken
	}
	envs[fmt.Sprintf("%s_TOKEN", strings.ToUpper(config.GitProvider))] = tokenValue
	envs[fmt.Sprintf("%s_OWNER", strings.ToUpper(config.GitProvider))] = viper.GetString(fmt.Sprintf("flags.%s-owner", config.GitProvider))
	envs["TF_VAR_email_address"] = viper.GetString("flags.alerts-email")
	envs["TF_VAR_vault_addr"] = VaultPortForwardURL
	envs["TF_VAR_vault_token"] = readVaultTokenFromSecret(clientset, config)
	envs[fmt.Sprintf("TF_VAR_%s_token", config.GitProvider)] = tokenValue
	envs["VAULT_ADDR"] = VaultPortForwardURL
	envs["VAULT_TOKEN"] = readVaultTokenFromSecret(clientset, config)
	envs["TF_VAR_vultr_token"] = os.Getenv("VULTR_API_KEY")
	envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
	envs["TF_VAR_atlantis_repo_webhook_url"] = viper.GetString(fmt.Sprintf("%s.atlantis.webhook.url", config.GitProvider))
	envs["TF_VAR_kbot_ssh_private_key"] = viper.GetString("kbot.private-key")
	envs["TF_VAR_kbot_ssh_public_key"] = viper.GetString("kbot.public-key")
	envs["VULTR_API_KEY"] = os.Getenv("VULTR_API_KEY")
	envs["AWS_ACCESS_KEY_ID"] = viper.GetString("kubefirst.state-store-creds.access-key-id")
	envs["AWS_SECRET_ACCESS_KEY"] = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")
	envs["TF_VAR_aws_access_key_id"] = viper.GetString("kubefirst.state-store-creds.access-key-id")
	envs["TF_VAR_aws_secret_access_key"] = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")

	switch config.GitProvider {
	case "gitlab":
		envs["TF_VAR_owner_group_id"] = viper.GetString("flags.gitlab-owner-group-id")
	}

	return envs
}
