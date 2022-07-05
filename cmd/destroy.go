/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "destroy the kubefirst management cluster",
	Long: `destory the kubefirst management cluster
and all of the components in kubernetes.

Optional: skip gitlab terraform 
if the registry has already been delteted.`,
	Run: func(cmd *cobra.Command, args []string) {

		kPortForward := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "gitlab", "port-forward", "svc/gitlab-webservice-default", "8888:8080")
		kPortForward.Stdout = os.Stdout
		kPortForward.Stderr = os.Stderr
		err := kPortForward.Start()
		defer kPortForward.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Panicf("error: failed to port-forward to gitlab %s", err)
		}
		// todo this needs to be removed when we are no longer in the starter account

		log.Println("\n\nTODO -- need to setup and argocd delete against registry and wait?\n\n")
		// kubeconfig := os.Getenv("HOME") + "/.kube/config"
		// config, err := argocdclientset.BuildConfigFromFlags("", kubeconfig)
		// argocdclientset, err := argocdclientset.NewForConfig(config)
		// if err != nil {
		// 	return nil, err
		// }

		//* should we git clone the gitops repo when destroy is run back to their
		//* local host to get the latest values of gitops

		os.Setenv("AWS_REGION", viper.GetString("aws.region"))
		os.Setenv("AWS_ACCOUNT_ID", viper.GetString("aws.accountid"))
		os.Setenv("HOSTED_ZONE_NAME", viper.GetString("aws.hostedzonename"))
		os.Setenv("GITLAB_TOKEN", viper.GetString("gitlab.token"))

		os.Setenv("TF_VAR_aws_account_id", viper.GetString("aws.accountid"))
		os.Setenv("TF_VAR_aws_region", viper.GetString("aws.region"))
		os.Setenv("TF_VAR_hosted_zone_name", viper.GetString("aws.hostedzonename"))

		directory := fmt.Sprintf("%s/.kubefirst/gitops/terraform/gitlab", home)
		skipGitlabTerraform, _ := cmd.Flags().GetBool("skip-gitlab-terraform")

		err = os.Chdir(directory)
		if err != nil {
			log.Panicf("error: could not change directory to " + directory)
		}

		os.Setenv("GITLAB_BASE_URL", "http://localhost:8888")

		if !skipGitlabTerraform {
			tfInitGitlabCmd := exec.Command(terraformPath, "init")
			tfInitGitlabCmd.Stdout = os.Stdout
			tfInitGitlabCmd.Stderr = os.Stderr
			err = tfInitGitlabCmd.Run()
			if err != nil {
				log.Panicf("failed to terraform init gitlab %s", err)
			}

			tfDestroyGitlabCmd := exec.Command(terraformPath, "destroy", "-auto-approve")
			tfDestroyGitlabCmd.Stdout = os.Stdout
			tfDestroyGitlabCmd.Stderr = os.Stderr
			err = tfDestroyGitlabCmd.Run()
			if err != nil {
				log.Panicf("failed to terraform destroy gitlab %s", err)
			}

			viper.Set("destroy.terraformdestroy.gitlab", true)
			viper.WriteConfig()
		}

		directory = fmt.Sprintf("%s/.kubefirst/gitops/terraform/base", home)
		err = os.Chdir(directory)
		if err != nil {
			log.Panicf("error: could not change directory to " + directory)
		}

		// delete argocd registry
		deleteRegistryApplication()
		log.Println("sleeping for 42 seconds to allow the registry to delete")
		time.Sleep(42 * time.Second)

		tfInitBaseCmd := exec.Command(terraformPath, "init")
		tfInitBaseCmd.Stdout = os.Stdout
		tfInitBaseCmd.Stderr = os.Stderr
		err = tfInitBaseCmd.Run()
		if err != nil {
			log.Panicf("failed to terraform init base %s", err)
		}

		tfDestroyBaseCmd := exec.Command(terraformPath, "destroy", "-auto-approve")
		tfDestroyBaseCmd.Stdout = os.Stdout
		tfDestroyBaseCmd.Stderr = os.Stderr
		err = tfDestroyBaseCmd.Run()
		if err != nil {
			log.Panicf("failed to terraform destroy base %s", err)
		}

		viper.Set("destroy.terraformdestroy.base", true)
		viper.WriteConfig()
	},
}

func init() {
	nebulousCmd.AddCommand(destroyCmd)

	destroyCmd.Flags().Bool("skip-gitlab-terraform", false, "whether to skip the terraform destroy against gitlab - note: if you already deleted registry it doesnt exist")
}


func deleteRegistryApplication() {
	log.Println("starting port forward to argocd server and deleting registry")
	kPortForward := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "argocd", "port-forward", "svc/argocd-server", "8080:8080")
	kPortForward.Stdout = os.Stdout
	kPortForward.Stderr = os.Stderr
	err := kPortForward.Start()
	defer kPortForward.Process.Signal(syscall.SIGTERM)
	if err != nil {
		log.Panicf("error: failed to port-forward to argocd %s", err)
	}

	url := "https://localhost:8080/api/v1/applications/registry"
	argoCdAppSync := exec.Command("curl", "-k", "-vL", "-X", "DELETE", url, "-H", fmt.Sprintf("Authorization: Bearer %s", viper.GetString("argocd.admin.apitoken")))
	argoCdAppSync.Stdout = os.Stdout
	argoCdAppSync.Stderr = os.Stderr
	err = argoCdAppSync.Run()
	if err != nil {
		log.Panicf("error: curl appSync failed failed %s", err)
	}
	log.Println("deleting argocd application registry")
}