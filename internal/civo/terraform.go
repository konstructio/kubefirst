package civo

import (
	"os"

	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/vault"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func readVaultTokenFromSecret() string {
	existingKubernetesSecret, err := k8s.ReadSecretV2(viper.GetString("k1-paths.kubeconfig"), vault.VaultNamespace, vault.VaultSecretName)
	if err != nil || existingKubernetesSecret == nil {
		log.Printf("Error reading existing Secret data: %s", err)
		return ""
	}

	return existingKubernetesSecret["root-token"]
}

func GetCivoTerraformEnvs(envs map[string]string) map[string]string {

	envs["CIVO_TOKEN"] = os.Getenv("CIVO_TOKEN")
	// needed for s3 api connectivity to object storage
	envs["AWS_ACCESS_KEY_ID"] = viper.GetString("kubefirst.state-store-creds.access-key-id")
	envs["AWS_SECRET_ACCESS_KEY"] = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")
	envs["TF_VAR_aws_access_key_id"] = viper.GetString("kubefirst.state-store-creds.access-key-id")
	envs["TF_VAR_aws_secret_access_key"] = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")

	return envs
}

func GetGithubTerraformEnvs(envs map[string]string) map[string]string {

	envs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
	envs["GITHUB_OWNER"] = viper.GetString("github.owner")
	// todo, this variable is assicated with repos.tf in gitops-template, considering bootstrap container image for metaphor
	// envs["TF_VAR_github_token"] = os.Getenv("GITHUB_TOKEN")
	envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("components.atlantis.webhook.secret")
	envs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kbot.public-key")
	envs["AWS_ACCESS_KEY_ID"] = viper.GetString("kubefirst.state-store-creds.access-key-id")
	envs["AWS_SECRET_ACCESS_KEY"] = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")
	envs["TF_VAR_aws_access_key_id"] = viper.GetString("kubefirst.state-store-creds.access-key-id")
	envs["TF_VAR_aws_secret_access_key"] = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")

	return envs
}

func GetUsersTerraformEnvs(envs map[string]string) map[string]string {

	envs["VAULT_TOKEN"] = readVaultTokenFromSecret()
	envs["VAULT_ADDR"] = viper.GetString("components.vault.port-forward-url")
	envs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
	envs["GITHUB_OWNER"] = viper.GetString("github.owner")

	return envs
}

func GetVaultTerraformEnvs(envs map[string]string) map[string]string {

	envs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
	envs["GITHUB_OWNER"] = viper.GetString("github.owner")
	envs["TF_VAR_email_address"] = viper.GetString("flags.admin-email")
	envs["TF_VAR_github_token"] = os.Getenv("GITHUB_TOKEN")
	envs["TF_VAR_vault_addr"] = viper.GetString("components.vault.port-forward-url")
	envs["TF_VAR_vault_token"] = readVaultTokenFromSecret()
	envs["VAULT_ADDR"] = viper.GetString("components.vault.port-forward-url")
	envs["VAULT_TOKEN"] = readVaultTokenFromSecret()
	envs["TF_VAR_civo_token"] = os.Getenv("CIVO_TOKEN")
	envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("components.atlantis.webhook.secret")
	envs["TF_VAR_atlantis_repo_webhook_url"] = viper.GetString("components.atlantis.webhook.url")
	envs["TF_VAR_kubefirst_bot_ssh_private_key"] = viper.GetString("kbot.private-key")
	envs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kbot.public-key")

	return envs
}
