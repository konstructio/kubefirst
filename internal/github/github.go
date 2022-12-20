package github

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

func ApplyGitHubTerraform(dryRun bool) {

	config := configs.ReadConfig()

	log.Info().Msg("Executing ApplyGithubTerraform")
	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, ApplyGithubTerraform skipped.")
		return
	}
	//* AWS_SDK_LOAD_CONFIG=1
	//* https://registry.terraform.io/providers/hashicorp/aws/2.34.0/docs#shared-credentials-file
	envs := map[string]string{}
	envs["AWS_SDK_LOAD_CONFIG"] = "1"
	envs["AWS_REGION"] = viper.GetString("aws.region")
	aws.ProfileInjection(&envs)
	// Prepare for terraform gitlab execution
	envs["GITHUB_TOKEN"] = os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")
	envs["GITHUB_OWNER"] = viper.GetString("github.owner")
	envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("github.atlantis.webhook.secret")
	envs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("botPublicKey")

	directory := fmt.Sprintf("%s/gitops/terraform/github", config.K1FolderPath)

	err := os.Chdir(directory)
	if err != nil {
		log.Panic().Msgf("error: could not change directory to %s", directory)
	}
	err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init")
	if err != nil {
		log.Panic().Msgf("error: terraform init for github failed %s", err)
	}

	err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "apply", "-auto-approve")
	if err != nil {
		log.Panic().Msgf("error: terraform apply for github failed %s", err)
	}
	os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))
	viper.Set("github.terraformapplied.gitops", true)
	viper.Set("terraform.github.apply.complete", true)
	viper.WriteConfig()
}

func DestroyGitHubTerraform(dryRun bool) {

	config := configs.ReadConfig()

	log.Info().Msg("Executing DestroyGitHubTerraform")
	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, DestroyGitHubTerraform skipped.")
		return
	}
	//* AWS_SDK_LOAD_CONFIG=1
	//* https://registry.terraform.io/providers/hashicorp/aws/2.34.0/docs#shared-credentials-file
	envs := map[string]string{}
	envs["AWS_SDK_LOAD_CONFIG"] = "1"
	envs["AWS_REGION"] = viper.GetString("aws.region")
	aws.ProfileInjection(&envs)
	// Prepare for terraform gitlab execution
	envs["GITHUB_TOKEN"] = os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")
	envs["GITHUB_OWNER"] = viper.GetString("github.owner")
	envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("github.atlantis.webhook.secret")
	envs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("botPublicKey")

	directory := fmt.Sprintf("%s/gitops/terraform/github", config.K1FolderPath)
	err := os.Chdir(directory)
	if err != nil {
		log.Panic().Msgf("error: could not change directory to %s", directory)
	}
	err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init")
	if err != nil {
		log.Panic().Msgf("error: terraform init for github failed %s", err)
	}

	err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "destroy", "-auto-approve")
	if err != nil {
		log.Panic().Msgf("error: terraform destroy for github failed %s", err)
	}
	os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))
	viper.Set("github.terraformapplied.gitops", true)
	viper.Set("terraform.github.apply.complete", true)
	viper.WriteConfig()
}

// todo destroy
