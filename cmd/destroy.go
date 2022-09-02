package cmd

import (
	"bytes"
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/spf13/cobra"
	"log"
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
		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			log.Panic(err)
		}

		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			log.Println(err)
		}

		if globalFlags.SilentMode {
			informUser(
				"Silent mode enabled, most of the UI prints wont be showed. Please check the logs for more details.\n",
				globalFlags.SilentMode,
			)
		}

		progressPrinter.GetInstance()
		progressPrinter.SetupProgress(2, globalFlags.SilentMode)

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
		informUser("Open gitlab port-forward", globalFlags.SilentMode)
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
		informUser("Open argocd port-forward", globalFlags.SilentMode)
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
		informUser("Open vault port-forward", globalFlags.SilentMode)
		progressPrinter.IncrementTracker("step-prepare", 1)

		log.Println("destroying gitlab terraform")

		progressPrinter.AddTracker("step-destroy", "Destroy Cloud", 4)
		progressPrinter.IncrementTracker("step-destroy", 1)
		informUser("Destroying Gitlab", globalFlags.SilentMode)
		gitlab.DestroyGitlabTerraform(skipGitlabTerraform)
		progressPrinter.IncrementTracker("step-destroy", 1)

		log.Println("gitlab terraform destruction complete")
		log.Println("deleting registry application in argocd")

		// delete argocd registry
		informUser("Destroying Registry Application", globalFlags.SilentMode)
		k8s.DeleteRegistryApplication(skipDeleteRegistryApplication)
		progressPrinter.IncrementTracker("step-destroy", 1)
		log.Println("registry application deleted")
		log.Println("terraform destroy base")
		informUser("Destroying Cluster", globalFlags.SilentMode)
		terraform.DestroyBaseTerraform(skipBaseTerraform)
		progressPrinter.IncrementTracker("step-destroy", 1)
		informUser("All Destroyed", globalFlags.SilentMode)

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
	destroyCmd.Flags().Bool("dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
	destroyCmd.Flags().Bool("silent", false, "enable silent mode will make the UI return less content to the screen")
	destroyCmd.Flags().Bool("use-telemetry", true, "installer will not send telemetry about this installation")

}
