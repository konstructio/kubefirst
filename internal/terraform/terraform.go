package terraform

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

func terraformConfig(terraformEntryPoint string) map[string]string {

	envs := map[string]string{}

	// todo lift common variables into this "global" space

	switch terraformEntryPoint {
	case "base":
		envs["AWS_PROFILE"] = viper.GetString("aws.profile")
		envs["TF_VAR_aws_account_id"] = viper.GetString("aws.accountid")
		envs["TF_VAR_aws_region"] = viper.GetString("aws.region")
		envs["TF_VAR_hosted_zone_name"] = viper.GetString("aws.hostedzonename")

		nodes_spot := viper.GetBool("aws.nodes_spot")
		if nodes_spot {
			envs["TF_VAR_lifecycle_nodes"] = "SPOT"
		}
	case "vault":
		fmt.Println("vault")
	case "gitlab":
		fmt.Println("gitlab")
	case "github":
		fmt.Println("github")
	case "users":
		//* AWS_SDK_LOAD_CONFIG=1
		//* https://registry.terraform.io/providers/hashicorp/aws/2.34.0/docs#shared-credentials-file
		envs["AWS_SDK_LOAD_CONFIG"] = "1"
		envs["AWS_PROFILE"] = viper.GetString("aws.profile")
		envs["TF_VAR_aws_region"] = viper.GetString("aws.region")
		envs["VAULT_TOKEN"] = viper.GetString("vault.token")
		envs["VAULT_ADDR"] = viper.GetString("vault.local.service")
		envs["GITHUB_TOKEN"] = os.Getenv("GITHUB_AUTH_TOKEN")
		envs["GITHUB_OWNER"] = viper.GetString("github.owner")
		return envs
	}
	return envs
}

func TerraformOutput(outputName string) {

}

func ApplyBaseTerraform(dryRun bool, directory string) {
	config := configs.ReadConfig()
	applyBase := viper.GetBool("create.terraformapplied.base")
	if applyBase != true {
		log.Println("Executing ApplyBaseTerraform")
		if dryRun {
			log.Printf("[#99] Dry-run mode, applyBaseTerraform skipped.")
			return
		}
		envs := map[string]string{}
		envs["AWS_PROFILE"] = viper.GetString("aws.profile")
		envs["TF_VAR_aws_account_id"] = viper.GetString("aws.accountid")
		envs["TF_VAR_aws_region"] = viper.GetString("aws.region")
		envs["TF_VAR_hosted_zone_name"] = viper.GetString("aws.hostedzonename")

		nodes_spot := viper.GetBool("aws.nodes_spot")
		if nodes_spot {
			envs["TF_VAR_lifecycle_nodes"] = "SPOT"
		}

		log.Println("tf env vars: ", envs)

		err := os.Chdir(directory)
		if err != nil {
			log.Panicf("error, directory does not exist - did you `kubefirst init`?: %s \nerror: %v", directory, err)
		}
		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "init")
		if err != nil {
			log.Panic(fmt.Sprintf("error: terraform init failed %v", err))
		}
		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "apply", "-auto-approve")
		if err != nil {
			log.Panic(fmt.Sprintf("error: terraform apply failed %v", err))
		}

		var terraformOutput bytes.Buffer
		k := exec.Command(config.TerraformPath, "output", "vault_unseal_kms_key")
		k.Stdout = &terraformOutput
		k.Stderr = os.Stderr
		errKey := k.Run()
		if errKey != nil {
			log.Panicf("error: terraform apply failed %v", errKey)
		}
		os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))
		keyIdNoSpace := strings.TrimSpace(terraformOutput.String())
		keyId := keyIdNoSpace[1 : len(keyIdNoSpace)-1]
		log.Println("keyid is:", keyId)
		viper.Set("vault.kmskeyid", keyId)
		viper.Set("create.terraformapplied.base", true)
		viper.WriteConfig()
		pkg.Detokenize(fmt.Sprintf("%s/gitops", config.K1FolderPath))
	} else {
		log.Println("Skipping: ApplyBaseTerraform")
	}
}

func DestroyBaseTerraform(skipBaseTerraform bool) {
	config := configs.ReadConfig()
	if !skipBaseTerraform {
		directory := fmt.Sprintf("%s/gitops/terraform/base", config.K1FolderPath)
		err := os.Chdir(directory)
		if err != nil {
			log.Panicf("error: could not change directory to " + directory)
		}

		envs := map[string]string{}
		envs["AWS_PROFILE"] = viper.GetString("aws.profile")
		envs["TF_VAR_aws_account_id"] = viper.GetString("aws.accountid")
		envs["TF_VAR_aws_region"] = viper.GetString("aws.region")
		envs["TF_VAR_hosted_zone_name"] = viper.GetString("aws.hostedzonename")

		nodes_spot := viper.GetBool("aws.nodes_spot")
		if nodes_spot {
			envs["TF_VAR_capacity_type"] = "SPOT"
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "init")
		if err != nil {
			log.Panicf("failed to terraform init base %v", err)
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "destroy", "-auto-approve")
		if err != nil {
			log.Panicf("failed to terraform destroy base %v", err)
		}
		viper.Set("destroy.terraformdestroy.base", true)
		viper.WriteConfig()
	} else {
		log.Println("skip:  destroyBaseTerraform")
	}
}

func ApplyECRTerraform(dryRun bool, directory string) {

	config := configs.ReadConfig()

	if !viper.GetBool("create.terraformapplied.ecr") {
		log.Println("Executing applyECRTerraform")
		if dryRun {
			log.Printf("[#99] Dry-run mode, applyECRTerraform skipped.")
			return
		}

		//* AWS_SDK_LOAD_CONFIG=1
		//* https://registry.terraform.io/providers/hashicorp/aws/2.34.0/docs#shared-credentials-file
		envs := map[string]string{}
		envs["AWS_SDK_LOAD_CONFIG"] = "1"
		envs["AWS_PROFILE"] = viper.GetString("aws.profile")
		envs["TF_VAR_aws_region"] = viper.GetString("aws.region")

		directory = fmt.Sprintf("%s/gitops/terraform/ecr", config.K1FolderPath)
		err := os.Chdir(directory)
		if err != nil {
			log.Panic("error: could not change directory to " + directory)
		}
		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "init")
		if err != nil {
			log.Panicf("error: terraform init for ecr failed %s", err)
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "apply", "-auto-approve")
		if err != nil {
			log.Panicf("error: terraform apply for ecr failed %s", err)
		}
		os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))
		viper.Set("create.terraformapplied.ecr", true)
		viper.WriteConfig()
	} else {
		log.Println("Skipping: applyECRTerraform")
	}
}

func DestroyECRTerraform(skipECRTerraform bool) {
	config := configs.ReadConfig()
	if !skipECRTerraform {
		directory := fmt.Sprintf("%s/gitops/terraform/ecr", config.K1FolderPath)
		err := os.Chdir(directory)
		if err != nil {
			log.Panicf("error: could not change directory to " + directory)
		}

		envs := map[string]string{}
		envs["AWS_PROFILE"] = viper.GetString("aws.profile")

		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "init")
		if err != nil {
			log.Printf("[WARN]: failed to terraform init (destroy) ECR, was the ECR not created(check AWS)?: %s", err)
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "destroy", "-auto-approve")
		if err != nil {
			log.Printf("[WARN]: failed to terraform destroy ECR, was the ECR not created (check AWS)?: %s", err)
		}
		viper.Set("destroy.terraformdestroy.ecr", true)
		viper.WriteConfig()
	} else {
		log.Println("skip:  destroyBaseTerraform")
	}
}

func ApplyUsersTerraform(dryRun bool, directory string) {

	config := configs.ReadConfig()

	if !viper.GetBool("create.terraformapplied.users") {
		log.Println("Executing ApplyUsersTerraform")
		if dryRun {
			log.Printf("[#99] Dry-run mode, ApplyUsersTerraform skipped.")
			return
		}
		//* AWS_SDK_LOAD_CONFIG=1
		//* https://registry.terraform.io/providers/hashicorp/aws/2.34.0/docs#shared-credentials-file
		envs := map[string]string{}
		envs["AWS_SDK_LOAD_CONFIG"] = "1"
		envs["AWS_PROFILE"] = viper.GetString("aws.profile")
		envs["TF_VAR_aws_region"] = viper.GetString("aws.region")
		envs["VAULT_TOKEN"] = viper.GetString("vault.token")
		envs["VAULT_ADDR"] = viper.GetString("vault.local.service")
		envs["GITHUB_TOKEN"] = os.Getenv("GITHUB_AUTH_TOKEN")
		envs["GITHUB_OWNER"] = viper.GetString("github.owner")

		err := os.Chdir(directory)
		if err != nil {
			log.Panic("error: could not change directory to " + directory)
		}
		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "init")
		if err != nil {
			log.Panicf("error: terraform init for users failed %s", err)
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "apply", "-auto-approve")
		if err != nil {
			log.Panicf("error: terraform apply for users failed %s", err)
		}
		os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))
		viper.Set("create.terraformapplied.users", true)
		viper.WriteConfig()
	} else {
		log.Println("Skipping: ApplyUsersTerraform")
	}
}

func InitApplyAutoApprove(dryRun bool, directory, tfEntrypoint string) {

	config := configs.ReadConfig()
	log.Printf("Entered Apply%sTerraform", strings.Title(tfEntrypoint))

	kubefirstConfigPath := fmt.Sprintf("terraform.%s.apply.executed", tfEntrypoint)

	if !viper.GetBool(kubefirstConfigPath) {
		log.Printf("Executing Apply%sTerraform", strings.Title(tfEntrypoint))
		if dryRun {
			log.Printf("[#99] Dry-run mode, Apply%sTerraform skipped", strings.Title(tfEntrypoint))
			return
		}

		envs := terraformConfig(tfEntrypoint)
		log.Println("tf env vars: ", envs)

		err := os.Chdir(directory)
		if err != nil {
			log.Panic("error: could not change to directory " + directory)
		}
		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "init")
		if err != nil {
			log.Panicf("error: terraform init for %s failed %s", tfEntrypoint, err)
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformPath, "apply", "-auto-approve")
		if err != nil {
			log.Panicf("error: terraform apply -auto-approve for %s failed %s", tfEntrypoint, err)
		}
		os.RemoveAll(fmt.Sprintf("%s/.terraform/", directory))
		viper.Set(kubefirstConfigPath, true)
		viper.WriteConfig()
	} else {
		log.Printf("skipping Apply%sTerraform", strings.Title(tfEntrypoint))
	}
}

// todo need to write something that outputs -json type and can get multiple values
func OutputSingleValue(directory, outputName string) {
	config := configs.ReadConfig()

	// tfOutputCmd := exec.Command("terraform", "output", "vault_unseal_kms_key")
	// tfOutputCmd.Stdout = &out
	// tfOutputCmd.Stderr = os.Stderr
	// err = tfOutputCmd.Run()
	// if err != nil {
	// 	fmt.Println("failed to call tfInitCmd.Run(): %v", err)
	// }

	var tfOutput bytes.Buffer
	k8 := exec.Command(config.TerraformPath, "refresh")
	k8.Stdout = &tfOutput
	k8.Stderr = os.Stderr
	errKey := k8.Run()
	if errKey != nil {
		log.Panicf("error: terraform output %s failed %v", outputName, errKey)
	}

	k := exec.Command(config.TerraformPath, "output", outputName)

	k.Stdout = &tfOutput
	k.Stderr = os.Stderr
	errKey = k.Run()
	if errKey != nil {
		log.Panicf("error: terraform output %s failed %v", outputName, errKey)
	}
	os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))
	keyIdNoSpace := strings.TrimSpace(tfOutput.String())
	keyId := keyIdNoSpace[1 : len(keyIdNoSpace)-1]
	log.Println("keyid is:", keyId)
	viper.Set("test.test.vault.kmskeyid", keyId)
	// viper.Set("test.terraformapplied.base", true) // todo
	viper.WriteConfig()
}
