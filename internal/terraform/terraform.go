package terraform

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

func terraformConfig(terraformEntryPoint string) map[string]string {
	envs := map[string]string{}

	if viper.GetString("cloud") == "aws" {
		//* https://registry.terraform.io/providers/hashicorp/aws/2.34.0/docs#shared-credentials-file
		envs["AWS_SDK_LOAD_CONFIG"] = "1"
		aws.ProfileInjection(&envs)
		envs["TF_VAR_aws_region"] = viper.GetString("aws.region")
		envs["AWS_REGION"] = viper.GetString("aws.region")
	}

	switch terraformEntryPoint {
	case "base":
		log.Info().Msg("Collecting Base Vars")
		envs["TF_VAR_aws_account_id"] = viper.GetString("aws.accountid")
		envs["TF_VAR_hosted_zone_name"] = viper.GetString("aws.hostedzonename")

		nodesSpot := viper.GetBool("aws.nodes_spot")
		if nodesSpot {
			envs["TF_VAR_lifecycle_nodes"] = "SPOT"
		}

		nodes_graviton := viper.GetBool("aws.nodes_graviton")
		if nodes_graviton {
			envs["TF_VAR_ami_type"] = "AL2_ARM_64"
			envs["TF_VAR_instance_type"] = "t4g.medium"
		}

		return envs
	case "vault":
		log.Info().Msg("Collecting Vault Vars")
		if viper.GetString("cloud") == pkg.CloudK3d {
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
		envs["TF_VAR_github_token"] = os.Getenv("GITHUB_TOKEN")
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
		log.Info().Msg("gitlab")
		return envs
	case "github":
		log.Info().Msg("Collecting github Vars")
		envs["GITHUB_TOKEN"] = os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")
		envs["GITHUB_OWNER"] = viper.GetString("github.owner")
		envs["TF_VAR_atlantis_repo_webhook_secret"] = viper.GetString("github.atlantis.webhook.secret")
		envs["TF_VAR_atlantis_repo_webhook_url"] = viper.GetString("github.atlantis.webhook.url")
		envs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kubefirst.bot.public-key")

		// todo: add validation for localhost
		envs["TF_VAR_email_address"] = viper.GetString("admin-email")
		envs["TF_VAR_github_token"] = os.Getenv("GITHUB_TOKEN")
		envs["TF_VAR_vault_addr"] = viper.GetString("vault.local.service")
		envs["TF_VAR_vault_token"] = viper.GetString("vault.token")
		envs["VAULT_ADDR"] = viper.GetString("vault.local.service")
		envs["VAULT_TOKEN"] = viper.GetString("vault.token")

		return envs
	case "users":
		log.Info().Msg("Collecting users Vars")
		envs["VAULT_TOKEN"] = viper.GetString("vault.token")
		envs["VAULT_ADDR"] = viper.GetString("vault.local.service")
		envs["GITHUB_TOKEN"] = os.Getenv("GITHUB_TOKEN")
		envs["GITHUB_OWNER"] = viper.GetString("github.owner")
		return envs
	}
	return envs
}

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
	envs["TF_VAR_kubefirst_bot_ssh_public_key"] = viper.GetString("kubefirst.bot.public-key")

	return envs
}

func ApplyBaseTerraform(dryRun bool, directory string) {
	config := configs.ReadConfig()
	applyBase := viper.GetBool("create.terraformapplied.base")
	if applyBase != true {
		log.Info().Msg("Executing ApplyBaseTerraform")
		if dryRun {
			log.Printf("[#99] Dry-run mode, applyBaseTerraform skipped.")
			return
		}
		envs := map[string]string{}

		aws.ProfileInjection(&envs)

		envs["TF_VAR_aws_account_id"] = viper.GetString("aws.accountid")
		envs["TF_VAR_aws_region"] = viper.GetString("aws.region")
		envs["TF_VAR_hosted_zone_name"] = viper.GetString("aws.hostedzonename")

		nodesSpot := viper.GetBool("aws.nodes_spot")
		if nodesSpot {
			envs["TF_VAR_lifecycle_nodes"] = "SPOT"
		}

		nodesGraviton := viper.GetBool("aws.nodes_graviton")
		if nodesGraviton {
			envs["TF_VAR_ami_type"] = "AL2_ARM_64"
			envs["TF_VAR_instance_type"] = "t4g.medium"
		}

		//commenting to avoid log creds
		//log.Print("tf env vars: ", envs)

		err := os.Chdir(directory)
		if err != nil {
			log.Panic().Msgf("error, directory does not exist - did you `kubefirst init`?: %s error: %v", directory, err)
		}
		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init")
		if err != nil {
			log.Panic().Err(err).Msg("error: terraform init failed")
		}
		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "apply", "-auto-approve")
		if err != nil {
			log.Panic().Err(err).Msg("error: terraform init failed")
		}

		var terraformOutput bytes.Buffer
		k := exec.Command(config.TerraformClientPath, "output", "vault_unseal_kms_key")
		k.Stdout = &terraformOutput
		k.Stderr = os.Stderr
		errKey := k.Run()
		if errKey != nil {
			log.Panic().Err(err).Msg("error: terraform apply failed")
		}
		keyIdNoSpace := strings.TrimSpace(terraformOutput.String())
		keyId := keyIdNoSpace[1 : len(keyIdNoSpace)-1]
		log.Info().Msgf("keyid is: %s", keyId)
		viper.Set("vault.kmskeyid", keyId)

		var terraformNodeArnOutput bytes.Buffer
		k = exec.Command(config.TerraformClientPath, "output", "eks_node_role_arn")
		k.Stdout = &terraformNodeArnOutput
		k.Stderr = os.Stderr
		errKey = k.Run()
		if errKey != nil {
			log.Panic().Err(err).Msg("error: terraform output failed")
		}
		os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))
		nodeGroupArn := strings.TrimSpace(terraformNodeArnOutput.String())
		nodeGroupArn = nodeGroupArn[1 : len(nodeGroupArn)-1]
		log.Info().Msgf("nodeGroupArn is: %s", nodeGroupArn)
		viper.Set("aws.node-group-arn", nodeGroupArn)
		viper.Set("create.terraformapplied.base", true)
		viper.WriteConfig()
		pkg.Detokenize(fmt.Sprintf("%s/gitops", config.K1FolderPath))
	} else {
		log.Info().Msg("Skipping: ApplyBaseTerraform")
	}
}

func DestroyBaseTerraform(skipBaseTerraform bool) {
	config := configs.ReadConfig()
	if !skipBaseTerraform {
		directory := fmt.Sprintf("%s/gitops/terraform/base", config.K1FolderPath)
		err := os.Chdir(directory)
		if err != nil {
			log.Panic().Msgf("error: could not change directory to " + directory)
		}

		envs := map[string]string{}

		aws.ProfileInjection(&envs)

		envs["TF_VAR_aws_account_id"] = viper.GetString("aws.accountid")
		envs["TF_VAR_aws_region"] = viper.GetString("aws.region")
		envs["TF_VAR_hosted_zone_name"] = viper.GetString("aws.hostedzonename")

		nodesSpot := viper.GetBool("aws.nodes_spot")
		if nodesSpot {
			envs["TF_VAR_capacity_type"] = "SPOT"
		}
		nodesGraviton := viper.GetBool("aws.nodes_graviton")
		if nodesGraviton {
			envs["TF_VAR_ami_type"] = "AL2_ARM_64"
			envs["TF_VAR_instance_type"] = "t4g.medium"
		}

		// Please, let's keep this as insurance mechanism to prevent:
		// https://github.com/kubefirst/kubefirst/issues/1015
		// 50% of the time, this should be no-op as ELB should be already be removed, but sometimes ELB are still there.
		err = aws.DestroyLoadBalancerByName(viper.GetString("aws.elb.name"))
		if err != nil {
			log.Info().Msgf("Failed to destroy load balancer: %v", err)
		}
		//To allow destruction of SG to not holde destroy of base
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			time.Sleep(120 * time.Second)
			//destroy all found sg
			// Please, let's keep this as insurance mechanism to prevent:
			// https://github.com/kubefirst/kubefirst/issues/1015
			// Also
			// https://github.com/kubefirst/kubefirst/issues/1022
			for _, sg := range viper.GetStringSlice("aws.elb.sg") {
				for i := 0; i < 15; i++ {
					log.Info().Msgf("Trying to Security Group: %s", sg)
					err := aws.DestroySecurityGroupById(sg)
					if err != nil {
						log.Warn().Msgf("Failed to destroy security group: %v", err)
						if strings.Contains(fmt.Sprintf("%v", err), "has a dependent object") {
							log.Debug().Msgf("Security Group has dependent: %s", sg)
							time.Sleep(120 * time.Second)
						} else {
							// If it is a 404 return or something else the error is ignored by design for now.
							// We assume it was already removed before
							log.Debug().Msgf("Security Group failed to be removed, and error was ignored: %s, %v", sg, err)
							break
						}
					} else {
						log.Debug().Msgf("Security Group was removed as expected: %s", sg)
						break
					}

				}
			}
			wg.Done()
		}()
		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init")
		if err != nil {
			log.Panic().Err(err).Msg("failed to terraform init base")
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "destroy", "-auto-approve")
		if err != nil {
			log.Panic().Err(err).Msg("failed to terraform destroy base")
		}
		wg.Wait()
		viper.Set("destroy.terraformdestroy.base", true)
		viper.WriteConfig()
		log.Info().Msg("terraform base destruction complete")
	} else {
		log.Info().Msg("skip: destroyBaseTerraform")
	}
}

func ApplyECRTerraform(dryRun bool, directory string) {

	config := configs.ReadConfig()

	if !viper.GetBool("create.terraformapplied.ecr") {
		log.Info().Msg("Executing applyECRTerraform")
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
			log.Panic().Msgf("error: could not change directory to " + directory)
		}
		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init")
		if err != nil {
			log.Panic().Msgf("error: terraform init for ecr failed %s", err)
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "apply", "-auto-approve")
		if err != nil {
			log.Panic().Msgf("error: terraform apply for ecr failed %s", err)
		}
		os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))
		viper.Set("create.terraformapplied.ecr", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("Skipping: applyECRTerraform")
	}
}

func DestroyECRTerraform(skipECRTerraform bool) {
	config := configs.ReadConfig()
	if !skipECRTerraform {
		directory := fmt.Sprintf("%s/gitops/terraform/ecr", config.K1FolderPath)
		err := os.Chdir(directory)
		if err != nil {
			log.Panic().Msgf("error: could not change directory to " + directory)
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
		log.Info().Msg("skip:  destroyBaseTerraform")
	}
}

func initActionAutoApprove(dryRun bool, tfAction, tfEntrypoint string, tfEnvs map[string]string) error {

	config := configs.ReadConfig()
	log.Printf("initActionAutoApprove - action: %s entrypoint: %s", tfAction, tfEntrypoint)

	if dryRun {
		log.Printf("[#99] Dry-run mode, action: %s entrypoint: %s", tfAction, tfEntrypoint)
		return nil
	}

	err := os.Chdir(tfEntrypoint)
	if err != nil {
		log.Info().Msg("error: could not change to directory " + tfEntrypoint)
		return err
	}
	err = pkg.ExecShellWithVars(tfEnvs, config.TerraformClientPath, "init")
	if err != nil {
		log.Printf("error: terraform init for %s failed: %s", tfEntrypoint, err)
		return err
	}

	err = pkg.ExecShellWithVars(tfEnvs, config.TerraformClientPath, tfAction, "-auto-approve")
	if err != nil {
		log.Printf("error: terraform %s -auto-approve for %s failed %s", tfAction, tfEntrypoint, err)
		return err
	}
	os.RemoveAll(fmt.Sprintf("%s/.terraform/", tfEntrypoint))
	os.Remove(fmt.Sprintf("%s/.terraform.lock.hcl", tfEntrypoint))
	return nil
}

func initAndMigrateActionAutoApprove(dryRun bool, tfAction, tfEntrypoint string) {

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
	//commenting to avoid log creds
	//log.Print("tf env vars: ", envs)

	err := os.Chdir(tfEntrypoint)
	if err != nil {
		log.Panic().Msgf("error: could not change to directory " + tfEntrypoint)
	}

	err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init", "-migrate-state", "-force-copy")
	if err != nil {
		log.Panic().Msgf("error: terraform init for %s failed %s", tfEntrypoint, err)
	}

	err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, tfAction, "-auto-approve")
	if err != nil {
		log.Panic().Msgf("error: terraform %s -auto-approve for %s failed %s", tfAction, tfEntrypoint, err)
	}
	os.RemoveAll(fmt.Sprintf("%s/.terraform/", tfEntrypoint))
	os.Remove(fmt.Sprintf("%s/.terraform.lock.hcl", tfEntrypoint))
	viper.Set(kubefirstConfigPath, true)
	viper.WriteConfig()
}

func InitAndReconfigureActionAutoApprove(dryRun bool, tfAction string, tfEntrypoint string) error {

	config := configs.ReadConfig()
	tfEntrypointSplit := strings.Split(tfEntrypoint, "/")
	kubefirstConfigProperty := tfEntrypointSplit[len(tfEntrypointSplit)-1]
	log.Printf("Entered Init%s%sTerraform", strings.Title(tfAction), strings.Title(kubefirstConfigProperty))

	kubefirstConfigPath := fmt.Sprintf("terraform.%s.%s.complete", kubefirstConfigProperty, tfAction)

	log.Printf("Executing Init%s%sTerraform", strings.Title(tfAction), strings.Title(kubefirstConfigProperty))
	if dryRun {
		log.Printf("[#99] Dry-run mode, Init%s%sTerraform skipped", strings.Title(tfAction), strings.Title(kubefirstConfigProperty))
		return nil
	}

	envs := terraformConfig(kubefirstConfigProperty)
	//commenting to avoid log creds
	//log.Print("tf env vars: ", envs)

	err := os.Chdir(tfEntrypoint)
	if err != nil {
		log.Error().Err(err).Msgf("error: could not change to directory " + tfEntrypoint)
		return err
	}

	err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init", "-reconfigure")
	if err != nil {
		log.Error().Err(err).Msgf("error: terraform init for %s failed %s", tfEntrypoint, err)
		return err
	}

	err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, tfAction, "-auto-approve")
	if err != nil {
		log.Error().Err(err).Msgf("error: terraform %s -auto-approve for %s failed %s", tfAction, tfEntrypoint, err)
		return err
	}
	os.RemoveAll(fmt.Sprintf("%s/.terraform/", tfEntrypoint))
	os.Remove(fmt.Sprintf("%s/.terraform.lock.hcl", tfEntrypoint))
	viper.Set(kubefirstConfigPath, true)
	viper.WriteConfig()

	return nil
}

func InitMigrateApplyAutoApprove(dryRun bool, tfEntrypoint string) {
	tfAction := "apply"
	initAndMigrateActionAutoApprove(dryRun, tfAction, tfEntrypoint)
}

func InitApplyAutoApprove(dryRun bool, tfEntrypoint string, tfEnvs map[string]string) error {
	tfAction := "apply"
	err := initActionAutoApprove(dryRun, tfAction, tfEntrypoint, tfEnvs)
	if err != nil {
		return err
	}
	return nil
}

func InitDestroyAutoApprove(dryRun bool, tfEntrypoint string, tfEnvs map[string]string) error {
	tfAction := "destroy"
	err := initActionAutoApprove(dryRun, tfAction, tfEntrypoint, tfEnvs)
	if err != nil {
		return err
	}
	return nil
}
func InitReconfigureDestroyAutoApprove(dryRun bool, tfEntrypoint string) {
	tfAction := "destroy"
	// todo make this private and match the rest of the terraform function calls
	InitAndReconfigureActionAutoApprove(dryRun, tfAction, tfEntrypoint)
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
		log.Error().Err(err).Msg("failed to call tfOutputCmd.Run()")
	}

	log.Print("tfOutput is: ", tfOutput.String())
}

// ApplyUsersTerraform load environment variables into the host based on the git provider, change directory to the
// Terraform required modules, terraform init, terraform apply and clean terraform files.
// todo: break it into smaller functions with no dependencies in order to allow unit tests
func ApplyUsersTerraform(dryRun bool, directory string, gitProvider string) error {

	config := configs.ReadConfig()

	if viper.GetBool("create.terraformapplied.users") || dryRun {
		log.Info().Msg("skipping: ApplyUsersTerraform")
		return nil
	}

	if len(gitProvider) == 0 {
		return errors.New("git provider not provided, skipping terraform apply")
	}

	log.Info().Msg("Executing ApplyUsersTerraform")

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
