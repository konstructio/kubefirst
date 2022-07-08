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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var skipGitlabTerraform bool
var skipDeleteRegistryApplication bool
var skipBaseTerraform bool
var destroyBuckets bool

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "destroy the kubefirst management cluster",
	Long: `destory the kubefirst management cluster
and all of the components in kubernetes.

Optional: skip gitlab terraform 
if the registry has already been delteted.`,
	Run: func(cmd *cobra.Command, args []string) {

		// todo this needs to be removed when we are no longer in the starter account
		destroyGitlabTerraform()
		// delete argocd registry
		deleteRegistryApplication()
		destroyBaseTerraform()
		//TODO: Remove buckets? Opt-in flag
		destroyBucketsInUse()

	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)

	destroyCmd.PersistentFlags().BoolVar(&skipGitlabTerraform, "skip-gitlab-terraform", false, "whether to skip the terraform destroy against gitlab - note: if you already deleted registry it doesnt exist")
	destroyCmd.PersistentFlags().BoolVar(&skipDeleteRegistryApplication, "skip-delete-register", false, "whether to skip deletion of resgister application ")
	destroyCmd.PersistentFlags().BoolVar(&skipBaseTerraform, "skip-base-terraform", false, "whether to skip the terraform destroy against base install - note: if you already deleted registry it doesnt exist")
	destroyCmd.PersistentFlags().BoolVar(&destroyBuckets, "destroy-buckets", false, "remove created aws buckets, not empty buckets are not cleaned")
}

func deleteRegistryApplication() {
	if !skipDeleteRegistryApplication {
		log.Println("starting port forward to argocd server and deleting registry")
		kPortForward := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "argocd", "port-forward", "svc/argocd-server", "8080:80")
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
	} else {
		log.Println("skip:  deleteRegistryApplication")
	}
}
