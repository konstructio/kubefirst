package cmd

import (
	"fmt"
	"log"
	"syscall"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/terraform"
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
if the registry has already been deleted.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		config := configs.ReadConfig()

		skipGitlabTerraform, err := cmd.Flags().GetBool("skip-gitlab-terraform")
		if err != nil {
			log.Println(err)
			return err
		}
		skipDeleteRegistryApplication, err := cmd.Flags().GetBool("skip-delete-register")
		if err != nil {
			log.Println(err)
			return err
		}
		skipBaseTerraform, err := cmd.Flags().GetBool("skip-base-terraform")
		if err != nil {
			log.Println(err)
			return err
		}

		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			log.Println(err)
			return err
		}

		if globalFlags.SilentMode {
			informUser(
				"Silent mode enabled, most of the UI prints wont be showed. Please check the logs for more details.\n",
				globalFlags.SilentMode,
			)
		}

		progressPrinter.GetInstance()
		progressPrinter.SetupProgress(2, globalFlags.SilentMode)

		if globalFlags.DryRun {
			skipGitlabTerraform = true
			skipDeleteRegistryApplication = true
			skipBaseTerraform = true
		}
		progressPrinter.AddTracker("step-prepare", "Open Ports", 3)

		informUser("Open argocd port-forward", globalFlags.SilentMode)
		progressPrinter.IncrementTracker("step-prepare", 1)

		log.Println("destroying gitlab terraform")

		progressPrinter.AddTracker("step-destroy", "Destroy Cloud", 4)
		progressPrinter.IncrementTracker("step-destroy", 1)
		informUser("Destroying Gitlab", globalFlags.SilentMode)
		if !skipGitlabTerraform {
			kPortForward, _ := k8s.PortForward(globalFlags.DryRun, "gitlab", "svc/gitlab-webservice-default", "8888:8080")
			defer func() {
				if kPortForward != nil {
					log.Println("Closed argo port forward")
					_ = kPortForward.Process.Signal(syscall.SIGTERM)
				}
			}()
			informUser("Open gitlab port-forward", globalFlags.SilentMode)
			progressPrinter.IncrementTracker("step-prepare", 1)

			gitlab.DestroyGitlabTerraform(skipGitlabTerraform)
		}
		progressPrinter.IncrementTracker("step-destroy", 1)

		log.Println("gitlab terraform destruction complete")

		//This should wrapped into a function, maybe to move to: k8s.DeleteRegistryApplication
		if !skipDeleteRegistryApplication {
			kPortForwardArgocd, _ := k8s.PortForward(globalFlags.DryRun, "argocd", "svc/argocd-server", "8080:80")
			defer func() {
				if kPortForwardArgocd != nil {
					log.Println("Closed argocd port forward")
					_ = kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
				}
			}()
			informUser("Open argocd port-forward", globalFlags.SilentMode)
			progressPrinter.IncrementTracker("step-prepare", 1)
			log.Println("disabling ArgoCD auto sync")
			argoCDUsername := viper.GetString("argocd.admin.username")
			argoCDPassword := viper.GetString("argocd.admin.password")

			token, err := argocd.GetArgoCDToken(argoCDUsername, argoCDPassword)
			if err != nil {
				log.Println(err)
				return err
			}

			argoCDApplication, err := argocd.GetArgoCDApplication(token, "registry")
			if err != nil {
				log.Println(err)
				return err
			}

			// set empty syncPolicy (disable auto-sync)
			argoCDApplication.Spec.SyncPolicy = struct{}{}
			err = argocd.PutArgoCDApplication(token, argoCDApplication)
			if err != nil {
				log.Println(err)
				// Do nothing, only log until we fix how to disable autho-synch of registry.
				//return err
			}
			log.Println("deleting registry application in argocd")
			// delete argocd registry
			informUser("Destroying Registry Application", globalFlags.SilentMode)
			k8s.DeleteRegistryApplication(skipDeleteRegistryApplication)
			progressPrinter.IncrementTracker("step-destroy", 1)
			log.Println("registry application deleted")
		}

		// delete ECR when github
		informUser("Destroy ECR Repos", globalFlags.SilentMode)
		terraform.DestroyECRTerraform(false)

		log.Println("terraform destroy base")
		informUser("Destroying Cluster", globalFlags.SilentMode)
		terraform.DestroyBaseTerraform(skipBaseTerraform)
		progressPrinter.IncrementTracker("step-destroy", 1)
		informUser("All Destroyed", globalFlags.SilentMode)

		log.Println("terraform base destruction complete")
		fmt.Println("End of execution destroy")
		time.Sleep(time.Millisecond * 100)

		return nil
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
