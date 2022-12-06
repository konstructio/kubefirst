package cmd

import (
	"errors"
	"fmt"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
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

		// todo: wrap business logic into the handler
		if config.GitHubPersonalAccessToken == "" {

			httpClient := http.DefaultClient
			gitHubService := services.NewGitHubService(httpClient)
			gitHubHandler := handlers.NewGitHubHandler(gitHubService)
			gitHubAccessToken, err := gitHubHandler.AuthenticateUser()
			if err != nil {
				return err
			}

			if len(gitHubAccessToken) == 0 {
				return errors.New("unable to retrieve a GitHub token for the user")
			}

			err = os.Setenv("KUBEFIRST_GITHUB_AUTH_TOKEN", gitHubAccessToken)
			if err != nil {
				return errors.New("unable to set KUBEFIRST_GITHUB_AUTH_TOKEN")
			}

			// todo: set common way to load env. values (viper->struct->load-env)
			// todo: use viper file to load it, not load env. value
			if err := os.Setenv("KUBEFIRST_GITHUB_AUTH_TOKEN", gitHubAccessToken); err != nil {
				return err
			}
			log.Info().Msg("\nKUBEFIRST_GITHUB_AUTH_TOKEN set via OAuth")
		}

		err = pkg.UpdateTerraformS3BackendForLocalhostAddress()
		if err != nil {
			return err
		}

		// todo add progress bars to this

		//* step 1.1 - open port-forward to state store and vault
		// todo --skip-git-terraform

		k8s.LoopUntilPodIsReady(globalFlags.DryRun)

		// todo: remove it
		time.Sleep(20 * time.Second)

		//* step 1.3 - terraform destroy github
		githubTfApplied := viper.GetBool("terraform.github.apply.complete")
		if githubTfApplied {
			informUser("terraform destroying github resources", globalFlags.SilentMode)
			tfEntrypoint := config.GitOpsLocalRepoPath + "/terraform/github"
			forceDestroy := false
			err := terraform.InitAndReconfigureActionAutoApprove(globalFlags.DryRun, "destroy", tfEntrypoint)
			if err != nil {
				forceDestroy = true
				log.Warn().Msg("unable to destroy via terraform, using destroy force")
			}

			if forceDestroy {
				gitHubClient := githubWrapper.New()
				err = forceLocalDestroy(gitHubClient)
				if err != nil {
					return err
				}
			}

			informUser("successfully destroyed github resources", globalFlags.SilentMode)
		}

		//* step 2 - delete k3d cluster
		// this could be useful for us to chase down in eks and destroy everything
		// in the cloud / cluster minus eks to iterate from argocd forward
		// todo --skip-cluster-destroy
		informUser("deleting k3d cluster", globalFlags.SilentMode)
		err = k3d.DeleteK3dCluster()
		if err != nil {
			return err
		}
		informUser("k3d cluster deleted", globalFlags.SilentMode)
		informUser("be sure to run `kubefirst clean` before your next cloud provision", globalFlags.SilentMode)

		//* step 3 - clean local .k1 dir
		// err = cleanCmd.RunE(cmd, args)
		// if err != nil {
		// 	log.Println("Error running:", cleanCmd.Name())
		// 	return err
		// }

		fmt.Println("End of execution destroy")
		time.Sleep(time.Millisecond * 100)

		return nil
	},
}

func forceLocalDestroy(gitHubClient githubWrapper.GithubSession) error {

	owner := viper.GetString("github.owner")
	sshKeyId := viper.GetString("botpublickey")

	err := gitHubClient.RemoveRepo(owner, "gitops")
	if err != nil {
		return err
	}
	err = gitHubClient.RemoveRepo(owner, "metaphor")
	if err != nil {
		return err
	}
	err = gitHubClient.RemoveRepo(owner, "metaphor-go")
	if err != nil {
		return err
	}
	err = gitHubClient.RemoveRepo(owner, "metaphor-frontend")
	if err != nil {
		return err
	}

	//err = gitHubClient.RemoveSSHKey(sshKeyId)
	err = gitHubClient.RemoveSSHKeyByPublicKey(owner, sshKeyId)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	clusterCmd.AddCommand(destroyLocalGithubCmd)
	currentCommand := destroyLocalGithubCmd
	flagset.DefineGlobalFlags(currentCommand)
	flagset.DefineDestroyFlags(currentCommand)
}
