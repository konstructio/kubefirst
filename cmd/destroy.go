package cmd

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "destroy Kubefirst management cluster",
	Long:  "destroy all the resources installed via Kubefirst installer",
	RunE: func(cmd *cobra.Command, args []string) error {

		config := configs.ReadConfig()

		destroyFlags, err := flagset.ProcessDestroyFlags(cmd)
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

		if viper.GetString("cloud") == "k3d" {
			// todo add progress bars to this

			//* step 1.1 - open port-forward to state store and vault
			// todo --skip-git-terraform
			kPortForwardMinio, err := k8s.PortForward(globalFlags.DryRun, "minio", "svc/minio", "9000:9000")
			defer func() {
				err = kPortForwardMinio.Process.Signal(syscall.SIGTERM)
				if err != nil {
					log.Println("Error closing kPortForwardMinio")
				}
			}()
			kPortForwardVault, err := k8s.PortForward(globalFlags.DryRun, "vault", "svc/vault", "8200:8200")
			defer func() {
				err = kPortForwardVault.Process.Signal(syscall.SIGTERM)
				if err != nil {
					log.Println("Error closing kPortForwardVault")
				}
			}()

			//* step 1.2
			// usersTfApplied := viper.GetBool("terraform.users.apply.complete")
			// if usersTfApplied {
			// 	informUser("terraform destroying users resources", globalFlags.SilentMode)
			// 	tfEntrypoint := config.GitOpsRepoPath + "/terraform/users"
			// 	terraform.InitDestroyAutoApprove(globalFlags.DryRun, tfEntrypoint)
			// 	informUser("successfully destroyed users resources", globalFlags.SilentMode)
			// }

			//* step 1.3 - terraform destroy github
			githubTfApplied := viper.GetBool("terraform.github.apply.complete")
			if githubTfApplied {
				informUser("terraform destroying github resources", globalFlags.SilentMode)
				tfEntrypoint := config.GitOpsRepoPath + "/terraform/github"
				terraform.InitDestroyAutoApprove(globalFlags.DryRun, tfEntrypoint)
				informUser("successfully destroyed github resources", globalFlags.SilentMode)
			}

			//* step 2 - delete k3d cluster
			// this could be useful for us to chase down in eks and destroy everything
			// in the cloud / cluster minus eks to iterate from argocd forward
			// todo --skip-cluster-destroy
			informUser("deleting k3d cluster", globalFlags.SilentMode)
			k3d.DeleteK3dCluster()
			informUser("k3d cluster deleted", globalFlags.SilentMode)
			informUser("be sure to run `kubefirst clean` before your next cloud provision", globalFlags.SilentMode)

			//* step 3 - clean local .k1 dir
			// err = cleanCmd.RunE(cmd, args)
			// if err != nil {
			// 	log.Println("Error running:", cleanCmd.Name())
			// 	return err
			// }
			os.Exit(0)
		}

		progressPrinter.SetupProgress(2, globalFlags.SilentMode)

		if globalFlags.DryRun {
			destroyFlags.SkipGitlabTerraform = true
			destroyFlags.SkipDeleteRegistryApplication = true
			destroyFlags.SkipBaseTerraform = true
		}
		progressPrinter.AddTracker("step-prepare", "Open Ports", 3)

		informUser("Open argocd port-forward", globalFlags.SilentMode)
		progressPrinter.IncrementTracker("step-prepare", 1)

		log.Println("destroying gitlab terraform")

		progressPrinter.AddTracker("step-destroy", "Destroy Cloud", 4)
		progressPrinter.IncrementTracker("step-destroy", 1)
		informUser("Destroying Gitlab", globalFlags.SilentMode)
		if !destroyFlags.SkipGitlabTerraform {
			kPortForward, _ := k8s.PortForward(globalFlags.DryRun, "gitlab", "svc/gitlab-webservice-default", "8888:8080")
			defer func() {
				if kPortForward != nil {
					log.Println("Closed GitLab port forward")
					_ = kPortForward.Process.Signal(syscall.SIGTERM)
				}
			}()
			informUser("Open gitlab port-forward", globalFlags.SilentMode)
			progressPrinter.IncrementTracker("step-prepare", 1)

			gitlab.DestroyGitlabTerraform(destroyFlags.SkipGitlabTerraform)
		}
		progressPrinter.IncrementTracker("step-destroy", 1)

		log.Println("gitlab terraform destruction complete")

		//This should wrapped into a function, maybe to move to: k8s.DeleteRegistryApplication
		if !destroyFlags.SkipDeleteRegistryApplication {
			kPortForwardArgocd, _ := k8s.PortForward(globalFlags.DryRun, "argocd", "svc/argocd-server", "8080:80")
			defer func() {
				if kPortForwardArgocd != nil {
					log.Println("Closed argocd port forward")
					_ = kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
				}
			}()
			informUser("Open argocd port-forward", globalFlags.SilentMode)
			progressPrinter.IncrementTracker("step-prepare", 1)

			log.Println("deleting registry application in argocd")
			// delete argocd registry
			informUser("Destroying Registry Application", globalFlags.SilentMode)
			k8s.DeleteRegistryApplication(destroyFlags.SkipDeleteRegistryApplication)
			progressPrinter.IncrementTracker("step-destroy", 1)
			log.Println("registry application deleted")
		}

		log.Println("terraform destroy base")
		informUser("Destroying Cluster", globalFlags.SilentMode)
		terraform.DestroyBaseTerraform(destroyFlags.SkipBaseTerraform)
		progressPrinter.IncrementTracker("step-destroy", 1)

		// destroy hosted zone
		if destroyFlags.HostedZoneDelete {
			hostedZone := viper.GetString("aws.hostedzonename")
			awsHandler := handlers.NewAwsHandler(hostedZone, destroyFlags)
			err := awsHandler.HostedZoneDelete()
			if err != nil {
				// if error, just log it
				log.Println(err)
			}
		}

		informUser("All Destroyed", globalFlags.SilentMode)

		log.Println("terraform base destruction complete")
		fmt.Println("End of execution destroy")
		time.Sleep(time.Millisecond * 100)

		return nil
	},
}

func init() {
	clusterCmd.AddCommand(destroyCmd)
	currentCommand := destroyCmd
	flagset.DefineGlobalFlags(currentCommand)
	flagset.DefineDestroyFlags(currentCommand)
}
