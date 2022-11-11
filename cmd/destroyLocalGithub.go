/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// destroyLocalGithubCmd represents the destroyLocalGithub command
var destroyLocalGithubCmd = &cobra.Command{
	Use:   "destroy-local-github",
	Short: "A brief description of your command",
	Long:  `TDB`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("destroy-local-github called")
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
		log.Println(destroyFlags, config)

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
			log.Println("\nKUBEFIRST_GITHUB_AUTH_TOKEN set via OAuth")
		}

		// todo add progress bars to this

		//* step 1.1 - open port-forward to vault
		// todo --skip-git-terraform
		kPortForwardVault, err := k8s.PortForward(globalFlags.DryRun, "vault", "svc/vault", "8200:8200")
		defer func() {
			err = kPortForwardVault.Process.Signal(syscall.SIGTERM)
			if err != nil {
				log.Println("Error closing kPortForwardVault")
			}
		}()

		//* step 1.3 - open port-forward to state store and terraform destroy github
		kPortForwardMinio, err := k8s.PortForward(globalFlags.DryRun, "minio", "svc/minio", "9000:9000")
		time.Sleep(20 * time.Second)
		githubTfApplied := viper.GetBool("terraform.github.apply.complete")
		if githubTfApplied {
			informUser("terraform destroying github resources", globalFlags.SilentMode)
			tfEntrypoint := config.GitOpsRepoPath + "/terraform/github"
			terraform.InitDestroyAutoApprove(globalFlags.DryRun, tfEntrypoint)
			informUser("successfully destroyed github resources", globalFlags.SilentMode)
		}

		defer func() {
			err = kPortForwardMinio.Process.Signal(syscall.SIGTERM)
			if err != nil {
				log.Println("Error closing kPortForwardMinio")
			}
		}()

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

		fmt.Println("End of execution destroy")
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
