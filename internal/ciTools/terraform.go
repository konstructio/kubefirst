package ciTools

import (
	"fmt"
	"log"
	"os"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

func ApplyCITerraform(dryRun bool, bucketName string) {

	config := configs.ReadConfig()

	if !viper.GetBool("create.terraformapplied.ci") {
		log.Println("Executing applyCITerraform")
		if dryRun {
			log.Printf("[#99] Dry-run mode, applyCITerraform skipped.")
			return
		}

		//* AWS_SDK_LOAD_CONFIG=1
		//* https://registry.terraform.io/providers/hashicorp/aws/2.34.0/docs#shared-credentials-file
		envs := map[string]string{}
		envs["AWS_SDK_LOAD_CONFIG"] = "1"
		envs["AWS_PROFILE"] = viper.GetString("aws.profile")
		envs["TF_VAR_aws_region"] = viper.GetString("aws.region")
		envs["TF_VAR_bucket_ci"] = bucketName

		directory := fmt.Sprintf("%s/ci/terraform/base", config.K1FolderPath)
		err := os.Chdir(directory)
		if err != nil {
			log.Panic("error: could not change directory to " + directory)
		}
		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "init")
		if err != nil {
			log.Panicf("error: terraform init for ci failed %s", err)
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "apply", "-auto-approve")
		if err != nil {
			log.Panicf("error: terraform apply for ci failed %s", err)
		}
		os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))

	} else {
		log.Println("Skipping: applyCITerraform")
	}
}

func DestroyCITerraform(skipCITerraform bool) {
	config := configs.ReadConfig()
	if !skipCITerraform {
		directory := fmt.Sprintf("%s/ci/terraform/base", config.K1FolderPath)
		err := os.Chdir(directory)
		if err != nil {
			log.Panicf("error: could not change directory to " + directory)
		}

		envs := map[string]string{}
		envs["AWS_PROFILE"] = viper.GetString("aws.profile")

		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "init")
		if err != nil {
			log.Printf("[WARN]: failed to terraform init (destroy) CI, was the CI not created(check AWS)?: %s", err)
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "destroy", "-auto-approve")
		if err != nil {
			log.Printf("[WARN]: failed to terraform destroy CI, was the CI not created (check AWS)?: %s", err)
		}
		viper.Set("destroy.terraformdestroy.ci", true)
		viper.WriteConfig()
	} else {
		log.Println("skip:  destroyBaseTerraform")
	}
}
