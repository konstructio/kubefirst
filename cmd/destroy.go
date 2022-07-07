package cmd

import (
	"github.com/kubefirst/nebulous/configs"
	"github.com/kubefirst/nebulous/internal/aws"
	"github.com/kubefirst/nebulous/internal/gitlab"
	"github.com/kubefirst/nebulous/internal/k8s"
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

		kPortForward := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "gitlab", "port-forward", "svc/gitlab-webservice-default", "8888:8080")
		kPortForward.Stdout = os.Stdout
		kPortForward.Stderr = os.Stderr
		err := kPortForward.Start()
		defer kPortForward.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Panicf("error: failed to port-forward to gitlab %s", err)
		}
		// todo this needs to be removed when we are no longer in the starter account
		gitlab.DestroyGitlabTerraform()
		// delete argocd registry
		k8s.DeleteRegistryApplication()
		destroyBaseTerraform()
		//TODO: Remove buckets? Opt-in flag
		aws.DestroyBucketsInUse()
	},
}

func init() {
	config := configs.ReadConfig()

	rootCmd.AddCommand(destroyCmd)

	destroyCmd.PersistentFlags().BoolVar(&config.SkipGitlabTerraform, "skip-gitlab-terraform", false, "whether to skip the terraform destroy against gitlab - note: if you already deleted registry it doesnt exist")
	destroyCmd.PersistentFlags().BoolVar(&config.SkipDeleteRegistryApplication, "skip-delete-register", false, "whether to skip deletion of resgister application ")
	destroyCmd.PersistentFlags().BoolVar(&config.SkipBaseTerraform, "skip-base-terraform", false, "whether to skip the terraform destroy against base install - note: if you already deleted registry it doesnt exist")
	destroyCmd.PersistentFlags().BoolVar(&config.DestroyBuckets, "destroy-buckets", false, "remove created aws buckets, not empty buckets are not cleaned")
}
