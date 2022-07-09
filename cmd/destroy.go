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
		kPortForwardGitlab := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "gitlab", "port-forward", "svc/gitlab-webservice-default", "8888:8080")
		kPortForwardGitlab.Stdout = os.Stdout
		kPortForwardGitlab.Stderr = os.Stderr
		err := kPortForwardGitlab.Start()
		defer kPortForwardGitlab.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Panicf("error: failed to port-forward to gitlab in main thread %s", err)
		}

		kPortForwardArgocd := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "argocd", "port-forward", "svc/argocd-server", "8080:80")
		kPortForwardArgocd.Stdout = os.Stdout
		kPortForwardArgocd.Stderr = os.Stderr
		err = kPortForwardArgocd.Start()
		defer kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Panicf("error: failed to port-forward to argocd in main thread %s", err)
		}
		kPortForwardVault := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "vault", "port-forward", "svc/vault", "8200:8200")
		kPortForwardVault.Stdout = os.Stdout
		kPortForwardVault.Stderr = os.Stderr
		err = kPortForwardVault.Start()
		defer kPortForwardVault.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Panicf("error: failed to port-forward to vault in main thread %s", err)
		}
		log.Println("detroying gitlab terraform")
		destroyGitlabTerraform()
		log.Println("gitlab terraform destruction complete")
		log.Println("deleting registry application in argocd")
		deleteArgocdRegistryApplication()
		log.Println("registry application deleted")
		log.Println("terraform destroy base")
		destroyBaseTerraform()
		log.Println("terraform base destruction complete")
		//TODO: move this step to `kubefirst clean` command and empty buckets and delete
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

func deleteArgocdRegistryApplication() {
	if !skipDeleteRegistryApplication {

		log.Println("refreshing argocd session token")
		getArgocdAuthToken()

		url := "https://localhost:8080/api/v1/applications/registry"
		argoCdAppSync := exec.Command("curl", "-k", "-vL", "-X", "DELETE", url, "-H", fmt.Sprintf("Authorization: Bearer %s", viper.GetString("argocd.admin.apitoken")))
		argoCdAppSync.Stdout = os.Stdout
		argoCdAppSync.Stderr = os.Stderr
		err := argoCdAppSync.Run()
		if err != nil {
			log.Panicf("error: delete registry applicatoin from argocd failed: %s", err)
		}
		log.Println("waiting for argocd deletion to complete")
		time.Sleep(240 * time.Second)
	} else {
		log.Println("skip:  deleteRegistryApplication")
	}
}
