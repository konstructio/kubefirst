package destroy

import (
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
	"syscall"
	"time"
)

var (
	dryRun              bool
	silentMode          bool
	hostedZoneDelete    bool
	skipBaseTerraform   bool
	skipDeleteRegister  bool
	hostedZoneKeepBase  bool
	skipGitlabTerraform bool
)

func NewCommand() *cobra.Command {

	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "destroy Kubefirst management cluster",
		Long:  "destroy all the resources installed via Kubefirst installer",
		RunE:  runDestroyCmd,
	}
	destroyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
	destroyCmd.Flags().BoolVar(&silentMode, "silent", false, "enable silent mode will make the UI return less content to the screen")
	destroyCmd.Flags().BoolVar(&hostedZoneDelete, "hosted-zone-delete", false, "delete full hosted zone, use --keep-base-hosted-zone in combination to keep base DNS records (NS, SOA, liveness)")
	destroyCmd.Flags().BoolVar(&skipBaseTerraform, "skip-base-terraform", false, "whether to skip the terraform destroy against base install - note: if you already deleted registry it doesnt exist")
	destroyCmd.Flags().BoolVar(&skipDeleteRegister, "skip-delete-register", false, "whether to skip deletion of register application")
	destroyCmd.Flags().BoolVar(&hostedZoneKeepBase, "hosted-zone-keep-base", false, "keeps base DNS records (NS, SOA and liveness TXT), and delete all other DNS records. Use it in combination with --hosted-zone-delete")
	destroyCmd.Flags().BoolVar(&skipGitlabTerraform, "skip-gitlab-terraform", false, "whether to skip the terraform destroy against gitlab - note: if you already deleted registry it doesnt exist")

	return destroyCmd
}

func runDestroyCmd(cmd *cobra.Command, args []string) error {

	config := configs.ReadConfig()

	if silentMode {
		pkg.InformUser(
			"Silent mode enabled, most of the UI prints wont be showed. Please check the logs for more details.\n",
			silentMode,
		)
	}

	if viper.GetString("cloud") == "k3d" {
		// todo add progress bars to this

		//* step 1.1 - open port-forward to state store and vault
		// todo --skip-git-terraform
		kPortForwardMinio, err := k8s.PortForward(dryRun, "minio", "svc/minio", "9000:9000")
		defer func() {
			err = kPortForwardMinio.Process.Signal(syscall.SIGTERM)
			if err != nil {
				log.Println("Error closing kPortForwardMinio")
			}
		}()
		kPortForwardVault, err := k8s.PortForward(dryRun, "vault", "svc/vault", "8200:8200")
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
			pkg.InformUser("terraform destroying github resources", silentMode)
			tfEntrypoint := config.GitOpsRepoPath + "/terraform/github"
			terraform.InitDestroyAutoApprove(dryRun, tfEntrypoint)
			pkg.InformUser("successfully destroyed github resources", silentMode)
		}

		//* step 2 - delete k3d cluster
		// this could be useful for us to chase down in eks and destroy everything
		// in the cloud / cluster minus eks to iterate from argocd forward
		// todo --skip-cluster-destroy
		pkg.InformUser("deleting k3d cluster", silentMode)
		k3d.DeleteK3dCluster()
		pkg.InformUser("k3d cluster deleted", silentMode)
		pkg.InformUser("be sure to run `kubefirst clean` before your next cloud provision", silentMode)

		//* step 3 - clean local .k1 dir
		// err = cleanCmd.RunE(cmd, args)
		// if err != nil {
		// 	log.Println("Error running:", cleanCmd.Name())
		// 	return err
		// }
		os.Exit(0)
	}

	progressPrinter.SetupProgress(2, silentMode)

	if dryRun {
		skipGitlabTerraform = true
		skipDeleteRegister = true
		skipBaseTerraform = true
	}
	progressPrinter.AddTracker("step-prepare", "Open Ports", 3)

	pkg.InformUser("Open argocd port-forward", silentMode)
	progressPrinter.IncrementTracker("step-prepare", 1)

	log.Println("destroying gitlab terraform")

	progressPrinter.AddTracker("step-destroy", "Destroy Cloud", 4)
	progressPrinter.IncrementTracker("step-destroy", 1)
	pkg.InformUser("Destroying Gitlab", silentMode)
	if !skipGitlabTerraform {
		kPortForward, _ := k8s.PortForward(dryRun, "gitlab", "svc/gitlab-webservice-default", "8888:8080")
		defer func() {
			if kPortForward != nil {
				log.Println("Closed GitLab port forward")
				_ = kPortForward.Process.Signal(syscall.SIGTERM)
			}
		}()
		pkg.InformUser("Open gitlab port-forward", silentMode)
		progressPrinter.IncrementTracker("step-prepare", 1)

		gitlab.DestroyGitlabTerraform(skipGitlabTerraform)
	}
	progressPrinter.IncrementTracker("step-destroy", 1)

	log.Println("gitlab terraform destruction complete")

	//This should wrapped into a function, maybe to move to: k8s.DeleteRegistryApplication
	if !skipDeleteRegister {
		kPortForwardArgocd, _ := k8s.PortForward(dryRun, "argocd", "svc/argocd-server", "8080:80")
		defer func() {
			if kPortForwardArgocd != nil {
				log.Println("Closed argocd port forward")
				_ = kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
			}
		}()
		pkg.InformUser("Open argocd port-forward", silentMode)
		progressPrinter.IncrementTracker("step-prepare", 1)

		log.Println("deleting registry application in argocd")
		// delete argocd registry
		pkg.InformUser("Destroying Registry Application", silentMode)
		k8s.DeleteRegistryApplication(skipDeleteRegister)
		progressPrinter.IncrementTracker("step-destroy", 1)
		log.Println("registry application deleted")
	}

	log.Println("terraform destroy base")
	pkg.InformUser("Destroying Cluster", silentMode)
	terraform.DestroyBaseTerraform(skipBaseTerraform)
	progressPrinter.IncrementTracker("step-destroy", 1)

	// destroy hosted zone
	if hostedZoneDelete {
		hostedZone := viper.GetString("aws.hostedzonename")
		awsHandler := handlers.NewAwsHandler(hostedZone, hostedZoneKeepBase)
		err := awsHandler.HostedZoneDelete()
		if err != nil {
			// if error, just log it
			log.Println(err)
		}
	}

	pkg.InformUser("All Destroyed", silentMode)

	log.Println("terraform base destruction complete")
	fmt.Println("End of execution destroy")
	time.Sleep(time.Millisecond * 100)

	return nil
}
