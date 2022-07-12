package terraform

import (
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	"log"
	"os"
	"strings"
	"bytes"
	"os/exec"
)

func ApplyBaseTerraform(dryRun bool, directory string) {
	config := configs.ReadConfig()
	applyBase := viper.GetBool("create.terraformapplied.base")
	if applyBase != true {
		log.Println("Executing ApplyBaseTerraform")
		if dryRun {
			log.Printf("[#99] Dry-run mode, applyBaseTerraform skipped.")
			return
		}
		os.Setenv("TF_VAR_aws_account_id", viper.GetString("aws.accountid"))
		os.Setenv("TF_VAR_aws_region", viper.GetString("aws.region"))
		os.Setenv("TF_VAR_hosted_zone_name", viper.GetString("aws.hostedzonename"))

		err := os.Chdir(directory)
		if err != nil {
			log.Panicf("error, directory does not exist - did you `kubefirst init`?: %s \nerror: %v", directory, err)
		}
		terraformInit := exec.Command(config.TerraformPath, "init")
		terraformInit.Stdout = os.Stdout
		terraformInit.Stderr = os.Stderr
		errInit := terraformInit.Run()
		if errInit != nil {
			log.Panic(fmt.Sprintf("error: terraform init failed %v", err))
		}
		terraformApply := exec.Command(config.TerraformPath, "apply", "-auto-approve")
		terraformApply.Stdout = os.Stdout
		terraformApply.Stderr = os.Stderr
		errApply := terraformApply.Run()
		if errApply != nil {
			log.Panic(fmt.Sprintf("error: terraform init failed %v", err))
		}
		
		var keyOut bytes.Buffer
		k := exec.Command(config.TerraformPath, "output", "vault_unseal_kms_key")
		k.Stdout = &keyOut
		k.Stderr = os.Stderr
		errKey := k.Run()
		if errKey != nil {
			log.Panicf("error: terraform apply failed %v", err)
		}
		os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))
		keyIdNoSpace := strings.TrimSpace(keyOut.String())
		keyId := keyIdNoSpace[1 : len(keyIdNoSpace)-1]
		log.Println("keyid is:", keyId)
		viper.Set("vault.kmskeyid", keyId)
		viper.Set("create.terraformapplied.base", true)
		viper.WriteConfig()
		pkg.Detokenize(fmt.Sprintf("%s/.kubefirst/gitops", config.HomePath))
	} else {
		log.Println("Skipping: ApplyBaseTerraform")
	}
}

func DestroyBaseTerraform(skipBaseTerraform bool) {
	config := configs.ReadConfig()
	if !skipBaseTerraform {
		directory := fmt.Sprintf("%s/.kubefirst/gitops/terraform/base", config.HomePath)
		err := os.Chdir(directory)
		if err != nil {
			log.Panicf("error: could not change directory to " + directory)
		}

		os.Setenv("TF_VAR_aws_account_id", viper.GetString("aws.accountid"))
		os.Setenv("TF_VAR_aws_region", viper.GetString("aws.region"))
		os.Setenv("TF_VAR_hosted_zone_name", viper.GetString("aws.hostedzonename"))

		terraformInit := exec.Command(config.TerraformPath, "init")
		terraformInit.Stdout = os.Stdout
		terraformInit.Stderr = os.Stderr
		errInit := terraformInit.Run()
		if errInit != nil {
			log.Panicf("failed to terraform init base %v", err)
		}

		terraformDestroy := exec.Command(config.TerraformPath, "destroy", "-auto-approve")
		terraformDestroy.Stdout = os.Stdout
		terraformDestroy.Stderr = os.Stderr
		errDestroy := terraformDestroy.Run()
		if errDestroy != nil {
			log.Panicf("failed to terraform destroy base %v", err)
		}
		viper.Set("destroy.terraformdestroy.base", true)
		viper.WriteConfig()
	} else {
		log.Println("skip:  destroyBaseTerraform")
	}
}
