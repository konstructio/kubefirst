package cmd

import (
	"fmt"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/wrappers"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// destroyLocalGithubCmd represents the destroyLocalGithub command
var destroyLocalGithubCmd = &cobra.Command{
	Use:   "destroy-local-github",
	Short: "A brief description of your command",
	Long:  `TDB`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config := configs.ReadConfig()

		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			log.Error().Err(err).Msg("")
			return err
		}

		if globalFlags.SilentMode {
			informUser(
				"Silent mode enabled, most of the UI prints wont be showed. Please check the logs for more details.\n",
				globalFlags.SilentMode,
			)
		}
		// silent is gold
		cmd.SilenceUsage = true

		log.Info().Msg("setting GitHub token...")
		httpClient := http.DefaultClient
		gitHubService := services.NewGitHubService(httpClient)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)
		_, err = wrappers.AuthenticateGitHubUserWrapper(config, gitHubHandler)
		if err != nil {
			return err
		}
		log.Info().Msg("GitHub token set!")

		log.Info().Msg("updating Terraform backend for localhost instead of minio...")
		err = pkg.UpdateTerraformS3BackendForLocalhostAddress()
		if err != nil {
			return err
		}
		log.Info().Msg("updating Terraform backend for localhost instead of minio, done")

		//* step 1.3 - terraform destroy github
		log.Info().Msg("running Terraform destroy...")
		githubTfApplied := viper.GetBool("terraform.github.apply.complete")
		if githubTfApplied {
			informUser("terraform destroying github resources", globalFlags.SilentMode)
			tfEntrypoint := config.GitOpsLocalRepoPath + "/terraform/github"
			forceDestroy := false
			err := terraform.InitAndReconfigureActionAutoApprove(globalFlags.DryRun, "destroy", tfEntrypoint)
			if err != nil {
				forceDestroy = true
				log.Warn().Msg("unable to destroy via terraform")
			} else {
				log.Info().Msg("running Terraform destroy, done")
			}

			if forceDestroy {
				log.Info().Msg("running force destroy...")
				gitHubClient := githubWrapper.New()
				err = pkg.ForceLocalDestroy(gitHubClient)
				if err != nil {
					return err
				}
				log.Info().Msg("force destroy, done")
			}

			informUser("successfully destroyed github resources", globalFlags.SilentMode)
		}

		// delete k3d cluster
		// todo --skip-cluster-destroy
		log.Info().Msg("deleting K3d cluster...")
		informUser("deleting k3d cluster", globalFlags.SilentMode)
		err = k3d.DeleteK3dCluster()
		if err != nil {
			return err
		}
		log.Info().Msg("deleting K3d cluster, done")
		informUser("k3d cluster deleted", globalFlags.SilentMode)

		informUser("be sure to run `kubefirst clean` before your next cloud provision", globalFlags.SilentMode)

		log.Info().Msg("end of execution destroy")
		fmt.Println("end of execution destroy")
		time.Sleep(time.Millisecond * 100)

		return nil
	},
}

func init() {
	clusterCmd.AddCommand(destroyLocalGithubCmd)
	currentCommand := destroyLocalGithubCmd
	flagset.DefineGlobalFlags(currentCommand)
	flagset.DefineDestroyFlags(currentCommand)
}
