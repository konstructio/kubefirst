package k3d

import (
	"os"

	"github.com/spf13/viper"
)

func GetGithubTerraformEnvs(envs map[string]string, envValues GithubTerraformEnvs) map[string]string {

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

type GithubTerraformEnvs struct {
	GithubToken           string
	GithubOwner           string
	AtlantisWebhookSecret string
	KbotSSHPublicKey      string
	AwsAccessKeyId        string
	AwsSecretAccessKey    string
}
