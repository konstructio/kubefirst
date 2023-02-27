package k3d

import (
	"os"

	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/vault"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func readVaultTokenFromSecret(config *K3dConfig) string {
	existingKubernetesSecret, err := k8s.ReadSecretV2(config.Kubeconfig, vault.VaultNamespace, vault.VaultSecretName)
	if err != nil || existingKubernetesSecret == nil {
		log.Printf("Error reading existing Secret data: %s", err)
		return ""
	}

	return existingKubernetesSecret["root-token"]
}

func GetGithubTerraformEnvs(envs map[string]string) map[string]string {

	envs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
	envs["GITHUB_OWNER"] = viper.GetString("flags.github-owner")
	// todo, this variable is assicated with repos.tf in gitops-template, considering bootstrap container image for metaphor
	envs["TF_VAR_github_token"] = os.Getenv("GITHUB_TOKEN")
	envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
	envs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kbot.public-key")
	envs["AWS_ACCESS_KEY_ID"] = viper.GetString("kubefirst.state-store-creds.access-key-id")
	envs["AWS_SECRET_ACCESS_KEY"] = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")
	envs["TF_VAR_aws_access_key_id"] = "kray"
	envs["TF_VAR_aws_secret_access_key"] = "feedkraystars"

	return envs
}

func GetUsersTerraformEnvs(config *K3dConfig, envs map[string]string) map[string]string {

	envs["VAULT_TOKEN"] = "k1_local_vault_token"
	envs["VAULT_ADDR"] = VaultPortForwardURL
	envs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")

	return envs
}

func GetVaultTerraformEnvs(config *K3dConfig, envs map[string]string) map[string]string {

	envs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
	envs["TF_VAR_email_address"] = "your@email.com"
	envs["TF_VAR_github_token"] = os.Getenv("GITHUB_TOKEN")
	envs["TF_VAR_vault_addr"] = VaultPortForwardURL
	envs["TF_VAR_vault_token"] = "k1_local_vault_token"
	envs["VAULT_ADDR"] = VaultPortForwardURL
	envs["VAULT_TOKEN"] = "k1_local_vault_token"
	envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("secrets.atlantis-webhook")
	envs["TF_VAR_aws_access_key_id"] = "kray"
	envs["TF_VAR_aws_secret_access_key"] = "feedkraystars"

	return envs
}

type GithubTerraformEnvs struct {
	GithubToken           string
	GithubOwner           string
	AtlantisWebhookSecret string
	KbotSSHPublicKey      string
	AwsAccessKeyId        string
	AwsSecretAccessKey    string
}
