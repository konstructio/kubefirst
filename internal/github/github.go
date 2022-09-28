package github

import (
	"fmt"
	"log"
	"os"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

func ApplyGitHubTerraform(dryRun bool, atlantisWebhookSecret string) {

	config := configs.ReadConfig()

	if !viper.GetBool("create.terraformapplied.github") {
		log.Println("Executing ApplyGithubTerraform")
		if dryRun {
			log.Printf("[#99] Dry-run mode, ApplyGithubTerraform skipped.")
			return
		}
		//* AWS_SDK_LOAD_CONFIG=1
		//* https://registry.terraform.io/providers/hashicorp/aws/2.34.0/docs#shared-credentials-file
		envs := map[string]string{}
		envs["AWS_SDK_LOAD_CONFIG"] = "1"
		envs["AWS_PROFILE"] = viper.GetString("aws.profile")
		// Prepare for terraform gitlab execution
		envs["GITHUB_TOKEN"] = viper.GetString("github.token")
		envs["GITHUB_OWNER"] = viper.GetString("github.owner")
		envs["TF_VAR_atlantis_repo_webhook_secret"] = atlantisWebhookSecret

		directory := fmt.Sprintf("%s/gitops/terraform/github", config.K1FolderPath)
		err := os.Chdir(directory)
		if err != nil {
			log.Panic("error: could not change directory to " + directory)
		}
		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "init")
		if err != nil {
			log.Panicf("error: terraform init for github failed %s", err)
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "apply", "-auto-approve")
		if err != nil {
			log.Panicf("error: terraform apply for github failed %s", err)
		}
		os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))
		viper.Set("create.terraformapplied.github", true)
		viper.WriteConfig()
	} else {
		log.Println("Skipping: ApplyGithubTerraform")
	}
}
