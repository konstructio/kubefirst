package cmd

import (
	"github.com/kubefirst/nebulous/configs"
	"github.com/kubefirst/nebulous/internal/aws"
	"github.com/kubefirst/nebulous/internal/gitlab"
	"github.com/kubefirst/nebulous/internal/k8s"
	"github.com/kubefirst/nebulous/internal/terraform"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/exec"
	"syscall"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "destroy the kubefirst management cluster",
	Long: `destory the kubefirst management cluster
and all of the components in k8s.

Optional: skip gitlab terraform 
if the registry has already been delteted.`,
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
			log.Panicf("error: failed to port-forward to gitlab %s", err)
		}
		// todo this needs to be removed when we are no longer in the starter account
		gitlab.DestroyGitlabTerraform(skipGitlabTerraform)
		// delete argocd registry
		k8s.DeleteRegistryApplication(skipDeleteRegistryApplication)
		terraform.DestroyBaseTerraform(skipBaseTerraform)
		//TODO: Remove buckets? Opt-in flag
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
