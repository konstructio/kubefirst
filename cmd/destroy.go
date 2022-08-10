package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"syscall"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/spf13/cobra"
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
		progressPrinter.GetInstance()
		progressPrinter.SetupProgress(2)

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

		// set profile
		profile, err := cmd.Flags().GetString("profile")
		if err != nil {
			log.Panicf("unable to get region values from viper")
		}
		viper.Set("aws.profile", profile)
		// propagate it to local environment
		err = os.Setenv("AWS_PROFILE", profile)
		if err != nil {
			log.Panicf("unable to set environment variable AWS_PROFILE, error is: %v", err)
		}
		log.Println("profile:", profile)

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

		if dryRun {
			skipGitlabTerraform = true
			skipDeleteRegistryApplication = true
			skipBaseTerraform = true
		}
		progressPrinter.AddTracker("step-prepare", "Open Ports", 3)

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
		informUser("Open gitlab port-forward")
		progressPrinter.IncrementTracker("step-prepare", 1)

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
		informUser("Open argocd port-forward")
		progressPrinter.IncrementTracker("step-prepare", 1)

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
		informUser("Open vault port-forward")
		progressPrinter.IncrementTracker("step-prepare", 1)

		log.Println("destroying gitlab terraform")

		progressPrinter.AddTracker("step-destroy", "Destroy Cloud", 4)
		progressPrinter.IncrementTracker("step-destroy", 1)
		informUser("Destroying Gitlab")
		gitlab.DestroyGitlabTerraform(skipGitlabTerraform)
		progressPrinter.IncrementTracker("step-destroy", 1)

		log.Println("gitlab terraform destruction complete")
		log.Println("deleting registry application in argocd")

		// delete argocd registry
		informUser("Destroying Registry Application")
		k8s.DeleteRegistryApplication(skipDeleteRegistryApplication)
		progressPrinter.IncrementTracker("step-destroy", 1)
		log.Println("registry application deleted")
		log.Println("terraform destroy base")
		informUser("Destroying Cluster")
		terraform.DestroyBaseTerraform(skipBaseTerraform)
		progressPrinter.IncrementTracker("step-destroy", 1)
		informUser("All Destroyed")

		log.Println("terraform base destruction complete")
		fmt.Println("End of execution destroy")
		time.Sleep(time.Millisecond * 100)
	},
}

func init() {

	rootCmd.AddCommand(destroyCmd)

	destroyCmd.Flags().Bool("skip-gitlab-terraform", false, "whether to skip the terraform destroy against gitlab - note: if you already deleted registry it doesnt exist")
	destroyCmd.Flags().Bool("skip-delete-register", false, "whether to skip deletion of register application ")
	destroyCmd.Flags().Bool("skip-base-terraform", false, "whether to skip the terraform destroy against base install - note: if you already deleted registry it doesnt exist")
	destroyCmd.Flags().Bool("destroy-buckets", false, "remove created aws buckets, not empty buckets are not cleaned")
	destroyCmd.Flags().Bool("dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")

	// AWS assume role
	destroyCmd.Flags().String("aws-assume-role", "", "instead of using AWS IAM user credentials, AWS AssumeRole feature generate role based credentials, more at https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html")
}
