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
		terraformAction := "apply"

		os.Setenv("TF_VAR_aws_account_id", viper.GetString("aws.accountid"))
		os.Setenv("TF_VAR_aws_region", viper.GetString("aws.region"))
		os.Setenv("TF_VAR_hosted_zone_name", viper.GetString("aws.domainname"))

		err := os.Chdir(directory)
		if err != nil {
			log.Panicf("error changing dir")
		}

		viperDestoryFlag := viper.GetBool("terraform.destroy")
		cmdDestroyFlag, _ := cmd.Flags().GetBool("destroy")

		if viperDestoryFlag == true || cmdDestroyFlag == true {
			terraformAction = "destroy"
		}

		log.Println("terraform action: ", terraformAction, "destroyFlag: ", viperDestoryFlag)
		execShellReturnStrings(terraformPath, "init")
		execShellReturnStrings(terraformPath, fmt.Sprintf("%s", terraformAction), "-auto-approve")
		keyOut, _, errKey := execShellReturnStrings(terraformPath, "output", "vault_unseal_kms_key")
		if errKey != nil {
			log.Panicf("failed to call tfOutputCmd.Run(): ", err)
		}
		keyId := strings.TrimSpace(keyOut)
		log.Println("keyid is:", keyId)
		viper.Set("vault.kmskeyid", keyId)
		viper.Set("create.terraformapplied.base", true)
		viper.WriteConfig()
		detokenize(fmt.Sprintf("%s/.kubefirst/gitops", home))
	} else {
		log.Println("Skipping: ApplyBaseTerraform")
	}
}