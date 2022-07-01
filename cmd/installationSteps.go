package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"os/exec"
	"syscall"
	"time"

	"net/url"
	"net/http"
	"encoding/json"
)

func applyBaseTerraform(cmd *cobra.Command,directory string){
	applyBase := viper.GetBool("create.terraformapplied.base")
	if applyBase != true {
		log.Println("Executing ApplyBaseTerraform")
		terraformAction := "apply"

		os.Setenv("TF_VAR_aws_account_id", viper.GetString("aws.accountid"))
		os.Setenv("TF_VAR_aws_region", viper.GetString("aws.region"))
		os.Setenv("TF_VAR_hosted_zone_name", viper.GetString("aws.domainname"))

		err := os.Chdir(directory)
		if err != nil {
			fmt.Println("error changing dir")
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
			fmt.Println("failed to call tfOutputCmd.Run(): ", err)
		}
		keyId := strings.TrimSpace(keyOut)
		fmt.Println("keyid is:", keyId)
		viper.Set("vault.kmskeyid", keyId)
		viper.Set("create.terraformapplied.base", true)
		viper.WriteConfig()
		detokenize(fmt.Sprintf("%s/.kubefirst/gitops", home))
	} else {
		log.Println("Skipping: ApplyBaseTerraform")
	}
}


func applyGitlabTerraform(directory string){
	if !viper.GetBool("create.terraformapplied.gitlab") {
		log.Println("Executing applyGitlabTerraform")
		// Prepare for terraform gitlab execution
		os.Setenv("GITLAB_TOKEN", viper.GetString("gitlab.token"))
		os.Setenv("GITLAB_BASE_URL", fmt.Sprintf("https://gitlab.%s", viper.GetString("aws.domainname")))

		directory = fmt.Sprintf("%s/.kubefirst/gitops/terraform/gitlab", home)
		err := os.Chdir(directory)
		if err != nil {
			fmt.Println("error changing dir")
		}
		execShellReturnStrings(terraformPath, "init")
		execShellReturnStrings(terraformPath, "apply", "-auto-approve")
		viper.Set("create.terraformapplied.gitlab", true)
		viper.WriteConfig()
	} else {
		log.Println("Skipping: applyGitlabTerraform")
	}
}

func configureSoftserveAndPush(){
	configureAndPushFlag := viper.GetBool("create.softserve.configure")
	if configureAndPushFlag != true {
		log.Println("Executing configureSoftserveAndPush")
		kPortForward := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "soft-serve", "port-forward", "svc/soft-serve", "8022:22")
		kPortForward.Stdout = os.Stdout
		kPortForward.Stderr = os.Stderr
		err := kPortForward.Start()
		defer kPortForward.Process.Signal(syscall.SIGTERM)
		if err != nil {
			fmt.Println("failed to call kPortForward.Run(): ", err)
		}
		time.Sleep(10 * time.Second)

		configureSoftServe()
		pushGitopsToSoftServe()
		viper.Set("create.softserve.configure", true)
		viper.WriteConfig()
		time.Sleep(10 * time.Second)
	} else {
		log.Println("Skipping: configureSoftserveAndPush")
	}
}

func gitlabKeyUpload(){
	// upload ssh public key
	if !viper.GetBool("gitlab.keyuploaded") {
		log.Println("Executing gitlabKeyUpload")
		log.Println("uploading ssh public key to gitlab")
		gitlabToken := viper.GetString("gitlab.token")
		data := url.Values{
			"title": {"kubefirst"},
			"key":   {viper.GetString("botpublickey")},
		}

		gitlabUrlBase := fmt.Sprintf("https://gitlab.%s", viper.GetString("aws.domainname"))

		resp, err := http.PostForm(gitlabUrlBase+"/api/v4/user/keys?private_token="+gitlabToken, data)
		if err != nil {
			log.Fatal(err)
		}
		var res map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&res)
		fmt.Println(res)
		fmt.Println("ssh public key uploaded to gitlab")
		viper.Set("gitlab.keyuploaded", true)
		viper.WriteConfig()
	} else {
		log.Println("Skipping: gitlabKeyUpload")
		log.Println("ssh public key already uploaded to gitlab")
	}
}