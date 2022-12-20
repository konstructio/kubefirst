/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// destroyAwsGithubCmd represents the destroyAwsGithub command
var destroyAwsGithubCmd = &cobra.Command{
	Use:   "destroy-aws-github",
	Short: "A brief description of your command",
	Long:  `TBD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("destroy-aws-github called")
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

		gitHubAccessToken := config.GitHubPersonalAccessToken
		httpClient := http.DefaultClient
		gitHubService := services.NewGitHubService(httpClient)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)

		if gitHubAccessToken == "" {

			gitHubAccessToken, err = gitHubHandler.AuthenticateUser()
			if err != nil {
				return err
			}

			if gitHubAccessToken == "" {
				return errors.New("cannot create a cluster without a github auth token. please export your " +
					"KUBEFIRST_GITHUB_AUTH_TOKEN in your terminal",
				)
			}

			// todo: set common way to load env. values (viper->struct->load-env)
			if err := os.Setenv("KUBEFIRST_GITHUB_AUTH_TOKEN", gitHubAccessToken); err != nil {
				return err
			}
			log.Println("\nKUBEFIRST_GITHUB_AUTH_TOKEN set via OAuth")
		}

		progressPrinter.SetupProgress(2, globalFlags.SilentMode)

		if globalFlags.DryRun {
			destroyFlags.SkipDeleteRegistryApplication = true
			destroyFlags.SkipBaseTerraform = true
			destroyFlags.SkipGithubTerraform = true
		}
		progressPrinter.AddTracker("step-prepare", "Open Ports", 3)

		progressPrinter.AddTracker("step-destroy", "Destroy Cloud", 4)
		progressPrinter.IncrementTracker("step-destroy", 1)

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

			informUser("Refreshing local gitops repository", globalFlags.SilentMode)
			log.Println("removing local gitops directory")
			os.RemoveAll(config.GitOpsRepoPath)

			log.Println("cloning fresh gitops directory from github owner's private gitops")
			gitClient.ClonePrivateRepo(fmt.Sprintf("https://github.com/%s/gitops", viper.GetString("github.owner")), config.GitOpsRepoPath)

			informUser("Removing ingress-nginx load balancer", globalFlags.SilentMode)

			log.Println("removing ingress-nginx.yaml from local gitops repo registry")
			os.Remove(fmt.Sprintf("%s/registry/ingress-nginx.yaml", config.GitOpsRepoPath))

			gitClient.PushLocalRepoUpdates("github.com", viper.GetString("github.owner"), "gitops", "origin")
			token, err := argocd.GetArgoCDToken("admin", viper.GetString("argocd.admin.password"))
			if err != nil {
				log.Fatal("could not collect argocd token", err)
			}

			log.Println("syncing argocd registry application")
			customTransport := http.DefaultTransport.(*http.Transport).Clone()
			customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			argocdHttpClient := http.Client{Transport: customTransport}
			log.Println("refreshing the registry application")
			argocd.RefreshApplication(&argocdHttpClient, "registry", token)
			log.Println("listing the applications after refresh, sleeping 15 seconds")
			argocd.ListApplications(&argocdHttpClient, "registry", token)

			time.Sleep(time.Second * 15)
			argocd.ListApplications(&argocdHttpClient, "registry", token)
			log.Println("listing the applications after 15 second sleep, syncing registry and sleeping 185 seconds")
			argocd.Sync(&argocdHttpClient, "registry", token)

			log.Println("waiting for nginx to deprovision load balancer and lb security groups")
			time.Sleep(time.Second * 185) // full 3 minutes poll + 5
			argocd.ListApplications(&argocdHttpClient, "registry", token)
			log.Println("listing the applications after 180 + 5 second sleep")

			log.Println("deleting registry application in argocd")
			// delete argocd registry
			informUser("Destroying Registry Application", globalFlags.SilentMode)
			k8s.DeleteRegistryApplication(destroyFlags.SkipDeleteRegistryApplication)
			log.Println("registry application deleted")
		}
		progressPrinter.IncrementTracker("step-destroy", 1)

		if !destroyFlags.SkipGithubTerraform {
			gitHubClient := githubWrapper.New()
			githubTfApplied := viper.GetBool("terraform.github.apply.complete")
			if githubTfApplied {
				informUser("terraform destroying github resources", globalFlags.SilentMode)
				tfEntrypoint := config.GitOpsRepoPath + "/terraform/github"
				forceDestroy := false
				err := terraform.InitAndReconfigureActionAutoApprove(globalFlags.DryRun, "destroy", tfEntrypoint)
				if err != nil {
					forceDestroy = true
					informUser("unable to destroy via terraform", globalFlags.SilentMode)
				} else {
					informUser("successfully destroyed github resources", globalFlags.SilentMode)
				}
				if forceDestroy {
					informUser("running force destroy...", globalFlags.SilentMode)
					err = pkg.ForceGithubDestroyCloud(gitHubClient)
					if err != nil {
						return err
					}
					informUser("force destroy, done", globalFlags.SilentMode)
				}
				//Best-effort basis, if terraform does its task, I believe it removes already.
				_ = pkg.GithubRemoveSSHKeys(gitHubClient)
			}
			log.Println("github terraform destruction complete")

		}
		progressPrinter.IncrementTracker("step-destroy", 1)

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

		fmt.Println("End of execution destroy")
		time.Sleep(time.Millisecond * 100)

		log.Println(destroyFlags, config)
		return nil
	},
}

func init() {
	clusterCmd.AddCommand(destroyAwsGithubCmd)
	currentCommand := destroyAwsGithubCmd
	flagset.DefineGlobalFlags(currentCommand)
	flagset.DefineDestroyFlags(currentCommand)
}
