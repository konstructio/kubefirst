package k3d

import (
	"os"

	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/vault"
	"github.com/rs/zerolog/log"
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
	envs["AWS_ACCESS_KEY_ID"] = "kray"
	envs["AWS_SECRET_ACCESS_KEY"] = "feedkraystars"
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
