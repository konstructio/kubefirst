package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func applyBaseTerraform(cmd *cobra.Command, directory string) {
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

		tfInitCmd := exec.Command(terraformPath, "init")
		tfInitCmd.Stdout = os.Stdout
		tfInitCmd.Stderr = os.Stderr
		err = tfInitCmd.Run()
		if err != nil {
			log.Panicf("error: terraform init for base failed %s", err)
		}

		tfApplyCmd := exec.Command(terraformPath, "apply", "-auto-approve")
		tfApplyCmd.Stdout = os.Stdout
		tfApplyCmd.Stderr = os.Stderr
		err = tfApplyCmd.Run()
		if err != nil {
			log.Panicf("error: terraform apply for base failed %s", err)
		}
		var outb bytes.Buffer
		tfOutputCmd := exec.Command(terraformPath, "output", "vault_unseal_kms_key")
		tfOutputCmd.Stdout = &outb
		tfOutputCmd.Stderr = os.Stderr
		err = tfOutputCmd.Run()
		if err != nil {
			log.Panicf("error: terraform apply for base output failed %s", err)
		}
		os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))
		keyIdNoSpace := strings.TrimSpace(outb.String())
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

func destroyBaseTerraform() {
	if !skipBaseTerraform {
		directory := fmt.Sprintf("%s/.kubefirst/gitops/terraform/base", home)
		err := os.Chdir(directory)
		if err != nil {
			log.Panicf("error: could not change directory to " + directory)
		}

		os.Setenv("TF_VAR_aws_account_id", viper.GetString("aws.accountid"))
		os.Setenv("TF_VAR_aws_region", viper.GetString("aws.region"))
		os.Setenv("TF_VAR_hosted_zone_name", viper.GetString("aws.hostedzonename"))

		tfInitCmd := exec.Command(terraformPath, "init")
		tfInitCmd.Stdout = os.Stdout
		tfInitCmd.Stderr = os.Stderr
		err = tfInitCmd.Run()
		if err != nil {
			log.Panicf("error: terraform init for destroy base failed %s", err)
		}

		tfApplyCmd := exec.Command(terraformPath, "destroy", "-auto-approve")
		tfApplyCmd.Stdout = os.Stdout
		tfApplyCmd.Stderr = os.Stderr
		err = tfApplyCmd.Run()
		if err != nil {
			log.Panicf("error: terraform destroy for base failed %s", err)
		}
		viper.Set("destroy.terraformdestroy.base", true)
		viper.WriteConfig()
	} else {
		log.Println("skip:  destroyBaseTerraform")
	}
}
