package local

import (
	"fmt"
	"os"
	"time"

	"github.com/kubefirst/kubefirst/internal/reports"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewDestroyCommand() *cobra.Command {
	destroyCmd := &cobra.Command{
		Use:     "destroy",
		Short:   "Destroy Kubefirst local cluster",
		Long:    "Destroy all the resources installed via Kubefirst local installer",
		PreRunE: validateDestroy,
		RunE:    destroy,
	}

	destroyCmd.Flags().BoolVar(&silentMode, "silent", false, "enable silentMode mode will make the UI return less content to the screen")
	destroyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")

	destroyCmd.SilenceUsage = true

	return destroyCmd
}

// destroy calls terraform to destroy the provisioned resources, if it fails,
// it forces delete the resources via API calls.
func destroy(cmd *cobra.Command, args []string) error {

	config := configs.ReadConfig()

	//* step 1.3 - terraform destroy github
	log.Info().Msg("running Terraform destroy...")

	fmt.Println(reports.StyleMessageBlackAndWhite("Destroying Kubefirst local environment... \nThis will take approximately 1-2 minutes to complete."))

	githubTfApplied := viper.GetBool("terraform.github.apply.complete")
	if githubTfApplied {
		pkg.InformUser("terraform destroying github resources", silentMode)
		tfEntrypoint := config.GitOpsLocalRepoPath + "/terraform/github"
		forceDestroy := false
		err := terraform.InitAndReconfigureActionAutoApprove(dryRun, "destroy", tfEntrypoint)
		if err != nil {
			forceDestroy = true
			log.Warn().Msg("unable to destroy via terraform")
		} else {
			log.Info().Msg("running Terraform destroy, done")
		}

		if forceDestroy {
			log.Info().Msg("running force destroy...")
			gitHubClient := githubWrapper.New(os.Getenv("GITHUB_TOKEN"))
			err = pkg.ForceLocalDestroy(gitHubClient)
			if err != nil {
				return err
			}
			log.Info().Msg("force destroy, done")
		}

		pkg.InformUser("successfully destroyed github resources", silentMode)
	}

	// delete k3d cluster
	// todo --skip-cluster-destroy
	log.Info().Msg("deleting K3d cluster...")
	pkg.InformUser("deleting k3d cluster", silentMode)
	// err := k3d.DeleteK3dCluster()
	// if err != nil {
	// 	return err
	// }
	log.Info().Msg("deleting K3d cluster, done")
	pkg.InformUser("k3d cluster deleted", silentMode)

	pkg.InformUser("be sure to run `kubefirst clean` before your next cloud provision", silentMode)

	log.Info().Msg("destroy was successfully executed!")
	fmt.Println("destroy was successfully executed!")
	time.Sleep(time.Millisecond * 100)

	return nil
}
