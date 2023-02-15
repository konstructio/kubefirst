package civo

import (
	"os"

	"github.com/spf13/viper"
)

func GetCivoTerraformEnvs(envs map[string]string) map[string]string {

	envs["CIVO_TOKEN"] = os.Getenv("CIVO_TOKEN")
	// needed for s3 api connectivity to object storage
	envs["AWS_ACCESS_KEY_ID"] = viper.GetString("civo.object-storage-creds.access-key-id")
	envs["AWS_SECRET_ACCESS_KEY"] = viper.GetString("civo.object-storage-creds.secret-access-key-id")
	envs["TF_VAR_aws_access_key_id"] = viper.GetString("civo.object-storage-creds.access-key-id")
	envs["TF_VAR_aws_secret_access_key"] = viper.GetString("civo.object-storage-creds.secret-access-key-id")

	return envs
}

func GetGithubTerraformEnvs(envs map[string]string) map[string]string {

	envs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
	envs["GITHUB_OWNER"] = viper.GetString("github.owner")
	// todo, this variable is assicated with repos.tf in gitops-template, considering bootstrap container image for metaphor
	// envs["TF_VAR_github_token"] = os.Getenv("GITHUB_TOKEN")
	envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("github.atlantis.webhook.secret")
	envs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kubefirst.bot.public-key")
	envs["AWS_ACCESS_KEY_ID"] = viper.GetString("civo.object-storage-creds.access-key-id")
	envs["AWS_SECRET_ACCESS_KEY"] = viper.GetString("civo.object-storage-creds.secret-access-key-id")
	envs["TF_VAR_aws_access_key_id"] = viper.GetString("civo.object-storage-creds.access-key-id")
	envs["TF_VAR_aws_secret_access_key"] = viper.GetString("civo.object-storage-creds.secret-access-key-id")

	return envs
}

func GetUsersTerraformEnvs(envs map[string]string) map[string]string {

	envs["VAULT_TOKEN"] = viper.GetString("vault.token")
	envs["VAULT_ADDR"] = viper.GetString("vault.local.service")
	envs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
	envs["GITHUB_OWNER"] = viper.GetString("github.owner")

	return envs
}

func GetVaultTerraformEnvs(envs map[string]string) map[string]string {

	envs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
	envs["GITHUB_OWNER"] = viper.GetString("github.owner")
	envs["TF_VAR_email_address"] = viper.GetString("admin-email")
	envs["TF_VAR_github_token"] = os.Getenv("GITHUB_TOKEN")
	envs["TF_VAR_vault_addr"] = viper.GetString("vault.local.service")
	envs["TF_VAR_vault_token"] = viper.GetString("vault.token")
	envs["VAULT_ADDR"] = viper.GetString("vault.local.service")
	envs["VAULT_TOKEN"] = viper.GetString("vault.token")
	envs["TF_VAR_civo_token"] = os.Getenv("CIVO_TOKEN")
	envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("github.atlantis.webhook.secret")
	envs["TF_VAR_atlantis_repo_webhook_url"] = viper.GetString("github.atlantis.webhook.url")
	envs["TF_VAR_kubefirst_bot_ssh_private_key"] = viper.GetString("kubefirst.bot.private-key")
	envs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kubefirst.bot.public-key")

	return envs
}
