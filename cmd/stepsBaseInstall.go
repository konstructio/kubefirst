package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func applyBaseTerraform(cmd *cobra.Command,directory string){
	applyBase := viper.GetBool("create.terraformapplied.base")
	if applyBase != true {
		log.Println("Executing ApplyBaseTerraform")
		if dryrunMode {
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
		_,_,errInit := execShellReturnStrings(terraformPath, "init")
		if errInit != nil {
			panic(fmt.Sprintf("error: terraform init failed %v", err))
		}
		_,_,errApply := execShellReturnStrings(terraformPath,"apply", "-auto-approve")
		if errApply != nil {
			panic(fmt.Sprintf("error: terraform init failed %v", err))
		}
		keyOut, _, errKey := execShellReturnStrings(terraformPath, "output", "vault_unseal_kms_key")
		if errKey != nil {
			log.Panicf("error: terraform apply failed %v", err)
		}
		os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))
		keyIdNoSpace :=  strings.TrimSpace(keyOut)
		keyId := keyIdNoSpace[1 : len(keyIdNoSpace)-1]
		log.Println("keyid is:", keyId)
		viper.Set("vault.kmskeyid", keyId)
		viper.Set("create.terraformapplied.base", true)
		viper.WriteConfig()
		detokenize(fmt.Sprintf("%s/.kubefirst/gitops", home))
	} else {
		log.Println("Skipping: ApplyBaseTerraform")
	}
}

func destroyBaseTerraform(){
	if !skipBaseTerraform {
		directory := fmt.Sprintf("%s/.kubefirst/gitops/terraform/base", home)
		err := os.Chdir(directory)
		if err != nil {
			log.Panicf("error: could not change directory to " + directory)
		}

		os.Setenv("TF_VAR_aws_account_id", viper.GetString("aws.accountid"))
		os.Setenv("TF_VAR_aws_region", viper.GetString("aws.region"))
		os.Setenv("TF_VAR_hosted_zone_name", viper.GetString("aws.hostedzonename"))

		_, _, errInit := execShellReturnStrings(terraformPath, "init")
		if errInit != nil {
			log.Panicf("failed to terraform init base %v", err)
		}

		_, _, errDestroy := execShellReturnStrings(terraformPath, "destroy", "-auto-approve")
		if errDestroy != nil {
			log.Panicf("failed to terraform destroy base %v", err)
		}
		viper.Set("destroy.terraformdestroy.base", true)
		viper.WriteConfig()
	} else {
		log.Println("skip:  destroyBaseTerraform")
	}
}
