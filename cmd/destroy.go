package cmd

import (
	"fmt"
	"github.com/kubefirst/nebulous/configs"
	"github.com/kubefirst/nebulous/internal/aws"
	"github.com/kubefirst/nebulous/internal/gitlab"
	"github.com/kubefirst/nebulous/internal/k8s"
	"github.com/kubefirst/nebulous/internal/terraform"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "destroy the kubefirst management cluster",
	Long: `destory the kubefirst management cluster
and all of the components in kubernetes.

Optional: skip gitlab terraform 
if the registry has already been deleted.`,
	Run: func(cmd *cobra.Command, args []string) {

		config := configs.ReadConfig()

		skipGitlabTerraform, err := cmd.Flags().GetBool("skip-gitlab-terraform")
		if err != nil {
			log.Panic(err)
		}
		skipDeleteRegistryApplication, err := cmd.Flags().GetBool("skip-delete-register")
		if err != nil {
			log.Panic(err)
		}
		skipBaseTerraform, err := cmd.Flags().GetBool("skip-base-terraform")
		if err != nil {
			log.Panic(err)
		}
		destroyBuckets, err := cmd.Flags().GetBool("destroy-buckets")
		if err != nil {
			log.Panic(err)
		}

		kPortForward := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "gitlab", "port-forward", "svc/gitlab-webservice-default", "8888:8080")
		kPortForward.Stdout = os.Stdout
		kPortForward.Stderr = os.Stderr
		defer kPortForward.Process.Signal(syscall.SIGTERM)
		err = kPortForward.Start()
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
		log.Println("destroying gitlab terraform")

		gitlab.DestroyGitlabTerraform(skipGitlabTerraform)
		log.Println("gitlab terraform destruction complete")
		log.Println("deleting registry application in argocd")

		// delete argocd registry
		k8s.DeleteRegistryApplication(skipDeleteRegistryApplication)
		log.Println("registry application deleted")
		log.Println("terraform destroy base")
		terraform.DestroyBaseTerraform(skipBaseTerraform)
		log.Println("terraform base destruction complete")
		//TODO: move this step to `kubefirst clean` command and empty buckets and delete
		aws.DestroyBucketsInUse(destroyBuckets)
	},
}

func init() {

	rootCmd.AddCommand(destroyCmd)

	destroyCmd.Flags().Bool("skip-gitlab-terraform", false, "whether to skip the terraform destroy against gitlab - note: if you already deleted registry it doesnt exist")
	destroyCmd.Flags().Bool("skip-delete-register", false, "whether to skip deletion of register application ")
	destroyCmd.Flags().Bool("skip-base-terraform", false, "whether to skip the terraform destroy against base install - note: if you already deleted registry it doesnt exist")
	destroyCmd.Flags().Bool("destroy-buckets", false, "remove created aws buckets, not empty buckets are not cleaned")
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
		time.Sleep(300 * time.Second)
	} else {
		log.Println("skip:  deleteRegistryApplication")
	}
}
