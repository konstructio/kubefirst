package github

import (
	"fmt"
	"log"
	"os"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

func ApplyGitHubTerraform(dryRun bool) {

	config := configs.ReadConfig()

	log.Println("Executing ApplyGithubTerraform")
	if dryRun {
		log.Printf("[#99] Dry-run mode, ApplyGithubTerraform skipped.")
		return
	}
	//* AWS_SDK_LOAD_CONFIG=1
	//* https://registry.terraform.io/providers/hashicorp/aws/2.34.0/docs#shared-credentials-file
	envs := map[string]string{}
	envs["AWS_SDK_LOAD_CONFIG"] = "1"
	aws.ProfileInjection(&envs)
	// Prepare for terraform gitlab execution
	envs["GITHUB_TOKEN"] = viper.GetString("github.token")
	envs["GITHUB_OWNER"] = viper.GetString("github.owner")
	envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("github.atlantis.webhook.secret")
	envs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("botPublicKey")

	directory := fmt.Sprintf("%s/gitops/terraform/github", config.K1FolderPath)

	err := os.Chdir(directory)
	if err != nil {
		log.Panic("error: could not change directory to " + directory)
	}
	err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init")
	if err != nil {
		log.Panicf("error: terraform init for github failed %s", err)
	}

	err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "apply", "-auto-approve")
	if err != nil {
		log.Panicf("error: terraform apply for github failed %s", err)
	}
	os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))
	viper.Set("github.terraformapplied.gitops", true)
	viper.WriteConfig()
}

func DestroyGitHubTerraform(dryRun bool) {

	config := configs.ReadConfig()

	log.Println("Executing DestroyGitHubTerraform")
	if dryRun {
		log.Printf("[#99] Dry-run mode, DestroyGitHubTerraform skipped.")
		return
	}
	//* AWS_SDK_LOAD_CONFIG=1
	//* https://registry.terraform.io/providers/hashicorp/aws/2.34.0/docs#shared-credentials-file
	envs := map[string]string{}
	envs["AWS_SDK_LOAD_CONFIG"] = "1"
	aws.ProfileInjection(&envs)
	// Prepare for terraform gitlab execution
	envs["GITHUB_TOKEN"] = viper.GetString("github.token")
	envs["GITHUB_OWNER"] = viper.GetString("github.owner")
	envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("github.atlantis.webhook.secret")
	envs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("botPublicKey")

	directory := fmt.Sprintf("%s/gitops/terraform/github", config.K1FolderPath)
	err := os.Chdir(directory)
	if err != nil {
		log.Panic("error: could not change directory to " + directory)
	}
	err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init")
	if err != nil {
		log.Panicf("error: terraform init for github failed %s", err)
	}

	err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "destroy", "-auto-approve")
	if err != nil {
		log.Panicf("error: terraform destroy for github failed %s", err)
	}
	os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))
	viper.Set("github.terraformapplied.gitops", true)
	viper.WriteConfig()
}

// todo destroy
