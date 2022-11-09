package terraform

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

func terraformConfig(terraformEntryPoint string) map[string]string {

	envs := map[string]string{}

	if viper.GetString("cloud") == "aws" {
		//* AWS_SDK_LOAD_CONFIG=1
		//* https://registry.terraform.io/providers/hashicorp/aws/2.34.0/docs#shared-credentials-file
		envs["AWS_SDK_LOAD_CONFIG"] = "1"
		aws.ProfileInjection(&envs)
		envs["TF_VAR_aws_region"] = viper.GetString("aws.region")
	}

	switch terraformEntryPoint {
	case "base":
		envs["TF_VAR_aws_account_id"] = viper.GetString("aws.accountid")
		envs["TF_VAR_hosted_zone_name"] = viper.GetString("aws.hostedzonename")

		nodes_spot := viper.GetBool("aws.nodes_spot")
		if nodes_spot {
			envs["TF_VAR_lifecycle_nodes"] = "SPOT"
		}
		return envs
	case "vault":

		if viper.GetString("cloud") == flagset.CloudLocal {
			envs["TF_VAR_email_address"] = viper.GetString("adminemail")
			envs["TF_VAR_github_token"] = os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")
			envs["TF_VAR_vault_addr"] = viper.GetString("vault.local.service")
			envs["TF_VAR_vault_token"] = viper.GetString("vault.token")
			envs["VAULT_ADDR"] = viper.GetString("vault.local.service")
			envs["VAULT_TOKEN"] = viper.GetString("vault.token")
			envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("github.atlantis.webhook.secret")
			envs["TF_VAR_atlantis_repo_webhook_url"] = viper.GetString("github.atlantis.webhook.url")
			envs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("botpublickey")
			return envs
		}

		envs["VAULT_ADDR"] = viper.GetString("vault.local.service")
		envs["VAULT_TOKEN"] = viper.GetString("vault.token")

		envs["AWS_SDK_LOAD_CONFIG"] = "1"
		aws.ProfileInjection(&envs)

		envs["AWS_DEFAULT_REGION"] = viper.GetString("aws.region")

		envs["TF_VAR_vault_addr"] = fmt.Sprintf("https://vault.%s", viper.GetString("aws.hostedzonename"))
		envs["TF_VAR_aws_account_id"] = viper.GetString("aws.accountid")
		envs["TF_VAR_aws_region"] = viper.GetString("aws.region")
		envs["TF_VAR_email_address"] = viper.GetString("adminemail")
		envs["TF_VAR_github_token"] = os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")
		envs["TF_VAR_hosted_zone_id"] = viper.GetString("aws.hostedzoneid") //# TODO: are we using this?
		envs["TF_VAR_hosted_zone_name"] = viper.GetString("aws.hostedzonename")
		envs["TF_VAR_vault_token"] = viper.GetString("vault.token")
		envs["TF_VAR_git_provider"] = viper.GetString("git.mode")
		//Escaping newline to allow certs to be loaded properly by terraform
		envs["TF_VAR_ssh_private_key"] = viper.GetString("botprivatekey")

		envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("github.atlantis.webhook.secret")
		envs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("botpublickey")
		return envs
	case "gitlab":
		fmt.Println("gitlab")
		return envs
	case "github":
		envs["GITHUB_TOKEN"] = os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")
		envs["GITHUB_OWNER"] = viper.GetString("github.owner")
		envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("github.atlantis.webhook.secret")
		envs["TF_VAR_atlantis_repo_webhook_url"] = viper.GetString("github.atlantis.webhook.url")
		envs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("botpublickey")

		// todo: add validation for localhost
		envs["TF_VAR_email_address"] = viper.GetString("adminemail")
		envs["TF_VAR_github_token"] = os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")
		envs["TF_VAR_vault_addr"] = viper.GetString("vault.local.service")
		envs["TF_VAR_vault_token"] = viper.GetString("vault.token")
		envs["VAULT_ADDR"] = viper.GetString("vault.local.service")
		envs["VAULT_TOKEN"] = viper.GetString("vault.token")

		return envs
	case "users":
		envs["VAULT_TOKEN"] = viper.GetString("vault.token")
		envs["VAULT_ADDR"] = viper.GetString("vault.local.service")
		envs["GITHUB_TOKEN"] = os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")
		envs["GITHUB_OWNER"] = viper.GetString("github.owner")
		return envs
	}
	return envs
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

		aws.ProfileInjection(&envs)

		envs["TF_VAR_aws_account_id"] = viper.GetString("aws.accountid")
		envs["TF_VAR_aws_region"] = viper.GetString("aws.region")
		envs["TF_VAR_hosted_zone_name"] = viper.GetString("aws.hostedzonename")

		nodes_spot := viper.GetBool("aws.nodes_spot")
		if nodes_spot {
			envs["TF_VAR_lifecycle_nodes"] = "SPOT"
		}

		nodes_graviton := viper.GetBool("aws.nodes_graviton")
		if nodes_graviton {
			envs["TF_VAR_ami_type"] = "AL2_ARM_64"
			envs["TF_VAR_instance_types"] = "m6g.medium"
		}

		log.Println("tf env vars: ", envs)

		err := os.Chdir(directory)
		if err != nil {
			log.Panicf("error, directory does not exist - did you `kubefirst init`?: %s \nerror: %v", directory, err)
		}
		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init")
		if err != nil {
			log.Panic(fmt.Sprintf("error: terraform init failed %v", err))
		}
		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "apply", "-auto-approve")
		if err != nil {
			log.Panic(fmt.Sprintf("error: terraform apply failed %v", err))
		}

		var terraformOutput bytes.Buffer
		k := exec.Command(config.TerraformClientPath, "output", "vault_unseal_kms_key")
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

		aws.ProfileInjection(&envs)

		envs["TF_VAR_aws_account_id"] = viper.GetString("aws.accountid")
		envs["TF_VAR_aws_region"] = viper.GetString("aws.region")
		envs["TF_VAR_hosted_zone_name"] = viper.GetString("aws.hostedzonename")

		nodes_spot := viper.GetBool("aws.nodes_spot")
		if nodes_spot {
			envs["TF_VAR_capacity_type"] = "SPOT"
		}
		nodes_graviton := viper.GetBool("aws.nodes_graviton")
		if nodes_graviton {
			envs["TF_VAR_ami_type"] = "AL2_ARM_64"
			envs["TF_VAR_instance_types"] = "m6g.medium"
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init")
		if err != nil {
			log.Panicf("failed to terraform init base %v", err)
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "destroy", "-auto-approve")
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

		aws.ProfileInjection(&envs)

		envs["TF_VAR_aws_region"] = viper.GetString("aws.region")

		directory = fmt.Sprintf("%s/gitops/terraform/ecr", config.K1FolderPath)
		err := os.Chdir(directory)
		if err != nil {
			log.Panic("error: could not change directory to " + directory)
		}
		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init")
		if err != nil {
			log.Panicf("error: terraform init for ecr failed %s", err)
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "apply", "-auto-approve")
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

		aws.ProfileInjection(&envs)

		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init")
		if err != nil {
			log.Printf("[WARN]: failed to terraform init (destroy) ECR, was the ECR not created(check AWS)?: %s", err)
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "destroy", "-auto-approve")
		if err != nil {
			log.Printf("[WARN]: failed to terraform destroy ECR, was the ECR not created (check AWS)?: %s", err)
		}
		viper.Set("destroy.terraformdestroy.ecr", true)
		viper.WriteConfig()
	} else {
		log.Println("skip:  destroyBaseTerraform")
	}
}

func initActionAutoApprove(dryRun bool, tfAction, tfEntrypoint string) {

	config := configs.ReadConfig()
	tfEntrypointSplit := strings.Split(tfEntrypoint, "/")
	kubefirstConfigProperty := tfEntrypointSplit[len(tfEntrypointSplit)-1]
	log.Printf("Entered Init%s%sTerraform", strings.Title(tfAction), strings.Title(kubefirstConfigProperty))

	kubefirstConfigPath := fmt.Sprintf("terraform.%s.%s.complete", kubefirstConfigProperty, tfAction)

	log.Printf("Executing Init%s%sTerraform", strings.Title(tfAction), strings.Title(kubefirstConfigProperty))
	if dryRun {
		log.Printf("[#99] Dry-run mode, Init%s%sTerraform skipped", strings.Title(tfAction), strings.Title(kubefirstConfigProperty))
	}

	envs := terraformConfig(kubefirstConfigProperty)
	log.Println("tf env vars: ", envs)

	err := os.Chdir(tfEntrypoint)
	if err != nil {
		log.Panic("error: could not change to directory " + tfEntrypoint)
	}
	err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init")
	if err != nil {
		log.Panicf("error: terraform init for %s failed %s", tfEntrypoint, err)
	}

	err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, tfAction, "-auto-approve")
	if err != nil {
		log.Panicf("error: terraform %s -auto-approve for %s failed %s", tfAction, tfEntrypoint, err)
	}
	os.RemoveAll(fmt.Sprintf("%s/.terraform/", tfEntrypoint))
	os.Remove(fmt.Sprintf("%s/.terraform.lock.hcl", tfEntrypoint))
	viper.Set(kubefirstConfigPath, true)
	viper.WriteConfig()
}

func InitApplyAutoApprove(dryRun bool, tfEntrypoint string) {
	tfAction := "apply"
	initActionAutoApprove(dryRun, tfAction, tfEntrypoint)
}

func InitDestroyAutoApprove(dryRun bool, tfEntrypoint string) {
	tfAction := "destroy"
	initActionAutoApprove(dryRun, tfAction, tfEntrypoint)
}

// todo need to write something that outputs -json type and can get multiple values
func OutputSingleValue(dryRun bool, directory, tfEntrypoint, outputName string) {

	config := configs.ReadConfig()
	os.Chdir(directory)

	var tfOutput bytes.Buffer
	tfOutputCmd := exec.Command(config.TerraformClientPath, "output", outputName)
	tfOutputCmd.Stdout = &tfOutput
	tfOutputCmd.Stderr = os.Stderr
	err := tfOutputCmd.Run()
	if err != nil {
		fmt.Println("failed to call tfOutputCmd.Run(): ", err)
	}

	log.Println("tfOutput is: ", tfOutput.String())
}

// ApplyUsersTerraform load environment variables into the host based on the git provider, change directory to the
// Terraform required modules, terraform init, terraform apply and clean terraform files.
// todo: break it into smaller functions with no dependencies in order to allow unit tests
func ApplyUsersTerraform(dryRun bool, directory string, gitProvider string) error {

	config := configs.ReadConfig()

	if viper.GetBool("create.terraformapplied.users") || dryRun {
		log.Println("skipping: ApplyUsersTerraform")
		return nil
	}

	if len(gitProvider) == 0 {
		return errors.New("git provider not provided, skipping terraform apply")
	}

	log.Println("Executing ApplyUsersTerraform")

	//* AWS_SDK_LOAD_CONFIG=1
	//* https://registry.terraform.io/providers/hashicorp/aws/2.34.0/docs#shared-credentials-file
	envs := map[string]string{}

	if gitProvider == "github" {
		envs["GITHUB_TOKEN"] = os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")
		envs["GITHUB_OWNER"] = viper.GetString("github.owner")
	} else if gitProvider == "gitlab" {
		envs["GITLAB_TOKEN"] = viper.GetString("gitlab.token")
		envs["GITLAB_BASE_URL"] = viper.GetString("gitlab.local.service")
	} else {
		return errors.New("a valid Git Provider wasn't provided, Terraform wasn't able to apply users")
	}

	envs["AWS_SDK_LOAD_CONFIG"] = "1"
	aws.ProfileInjection(&envs)
	envs["TF_VAR_aws_region"] = viper.GetString("aws.region")
	envs["VAULT_TOKEN"] = viper.GetString("vault.token")
	envs["VAULT_ADDR"] = viper.GetString("vault.local.service")
	envs["TF_VAR_initial_password"] = viper.GetString("botpassword")

	err := os.Chdir(directory)
	if err != nil {
		return fmt.Errorf("error: could not change directory to " + directory)
	}
	err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init")
	if err != nil {
		return fmt.Errorf("error: terraform init for users failed %s", err)
	}

	err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "apply", "-auto-approve")
	if err != nil {
		return fmt.Errorf("error: terraform apply for users failed %s", err)
	}
	err = os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))
	if err != nil {
		return err
	}

	// set that this step is successfully done and do not need to be called again
	viper.Set("create.terraformapplied.users", true)
	err = viper.WriteConfig()
	if err != nil {
		return err
	}
	return nil
}
