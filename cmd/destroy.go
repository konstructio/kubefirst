package cmd

import (
	"bytes"
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/spf13/cobra"
	"log"
	"os/exec"
	"syscall"
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

		arnRole, err := cmd.Flags().GetString("aws-assume-role")
		if err != nil {
			log.Println("unable to use the provided AWS IAM role for AssumeRole feature")
			return
		}

		if len(arnRole) > 0 {
			log.Println("calling assume role")
			err := aws.AssumeRole(arnRole)
			if err != nil {
				log.Println(err)
				return
			}
			log.Printf("assuming new AWS credentials based on role %q", arnRole)
		}

		var kPortForwardOutb, kPortForwardErrb bytes.Buffer
		kPortForward := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "gitlab", "port-forward", "svc/gitlab-webservice-default", "8888:8080")
		kPortForward.Stdout = &kPortForwardOutb
		kPortForward.Stderr = &kPortForwardErrb
		defer func() {
			_ = kPortForward.Process.Signal(syscall.SIGTERM)
		}()
		err = kPortForward.Start()
		if err != nil {
			log.Printf("warning: failed to port-forward to gitlab in main thread %s", err)
			log.Printf("Commad Execution STDOUT: %s", kPortForwardOutb.String())
			log.Printf("Commad Execution STDERR: %s", kPortForwardErrb.String())

		}

		if !skipDeleteRegistryApplication {
			var kPortForwardArgocdOutb, kPortForwardArgocdErrb bytes.Buffer
			kPortForwardArgocd := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "port-forward", "svc/argocd-server", "8080:80")
			kPortForwardArgocd.Stdout = &kPortForwardArgocdOutb
			kPortForwardArgocd.Stderr = &kPortForwardArgocdErrb
			err = kPortForwardArgocd.Start()
			defer func() {
				_ = kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
			}()
			if err != nil {
				log.Printf("error: failed to port-forward to argocd in main thread %s", err)
				log.Printf("Commad Execution STDOUT: %s", kPortForwardArgocdOutb.String())
				log.Printf("Commad Execution STDERR: %s", kPortForwardArgocdErrb.String())
			}
		}

		var kPortForwardVaultOutb, kPortForwardVaultErrb bytes.Buffer
		kPortForwardVault := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "vault", "port-forward", "svc/vault", "8200:8200")
		kPortForwardVault.Stdout = &kPortForwardVaultOutb
		kPortForwardVault.Stderr = &kPortForwardVaultErrb
		err = kPortForwardVault.Start()
		defer func() {
			_ = kPortForwardVault.Process.Signal(syscall.SIGTERM)
		}()
		if err != nil {
			log.Printf("error: failed to port-forward to vault in main thread %s", err)
			log.Printf("Commad Execution STDOUT: %s", kPortForwardVaultOutb.String())
			log.Printf("Commad Execution STDERR: %s", kPortForwardVaultErrb.String())
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
		fmt.Println("End of execution destroy")
	},
}

func init() {

	rootCmd.AddCommand(destroyCmd)

	destroyCmd.Flags().Bool("skip-gitlab-terraform", false, "whether to skip the terraform destroy against gitlab - note: if you already deleted registry it doesnt exist")
	destroyCmd.Flags().Bool("skip-delete-register", false, "whether to skip deletion of register application ")
	destroyCmd.Flags().Bool("skip-base-terraform", false, "whether to skip the terraform destroy against base install - note: if you already deleted registry it doesnt exist")
	destroyCmd.Flags().Bool("destroy-buckets", false, "remove created aws buckets, not empty buckets are not cleaned")

	// AWS assume role
	destroyCmd.Flags().String("aws-assume-role", "", "instead of using AWS IAM user credentials, AWS AssumeRole feature generate role based credentials, more at https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html")
}
