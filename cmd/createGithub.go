/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"syscall"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/helm"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/vault"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// createGithubCmd represents the createGithub command
var createGithubCmd = &cobra.Command{
	Use:   "create-github",
	Short: "create a kubefirst management cluster with github as Git Repo",
	Long:  `Create a kubefirst cluster using github as the Git Repo and setup integrations`,
	RunE: func(cmd *cobra.Command, args []string) error {

		config := configs.ReadConfig()
		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}

		progressPrinter.SetupProgress(4, globalFlags.SilentMode)

		//infoCmd need to be before the bars or it is printed in between bars:
		//Let's try to not move it on refactors
		infoCmd.Run(cmd, args)
		var kPortForwardArgocd *exec.Cmd
		progressPrinter.AddTracker("step-0", "Process Parameters", 1)
		progressPrinter.AddTracker("step-github", "Setup gitops on github", 3)
		progressPrinter.AddTracker("step-base", "Setup base cluster", 3)
		progressPrinter.AddTracker("step-apps", "Install apps to cluster", 6)

		progressPrinter.IncrementTracker("step-0", 1)

		if !globalFlags.UseTelemetry {
			informUser("Telemetry Disabled", globalFlags.SilentMode)
		}

		//* create github teams in the org and gitops repo
		informUser("Creating gitops/metaphor repos", globalFlags.SilentMode)
		err = githubAddCmd.RunE(cmd, args)
		if err != nil {
			log.Println("Error running:", githubAddCmd.Name())
			return err
		}

		informUser(fmt.Sprintf("Created GitOps Repo in github.com/%s", viper.GetString("github.owner")), globalFlags.SilentMode)
		progressPrinter.IncrementTracker("step-github", 1)

		//* push our locally detokenized gitops repo to remote github
		githubHost := viper.GetString("github.host")
		githubOwner := viper.GetString("github.owner")
		localRepo := "gitops"
		remoteName := "github"
		if !viper.GetBool("github.gitops.hydrated") {
			gitClient.PushLocalRepoToEmptyRemote(githubHost, githubOwner, localRepo, remoteName)
		} else {
			log.Println("already hydrated the github gitops repository")
		}

		progressPrinter.IncrementTracker("step-github", 1)

		//* push our locally detokenized gitops repo to remote github

		directory := fmt.Sprintf("%s/gitops/terraform/base", config.K1FolderPath)
		informUser("Creating K8S Cluster", globalFlags.SilentMode)
		terraform.ApplyBaseTerraform(globalFlags.DryRun, directory)
		progressPrinter.IncrementTracker("step-base", 1)

		// pushes detokenized KMS_KEY_ID
		if !viper.GetBool("vault.kms.kms-pushed") {
			gitClient.PushLocalRepoUpdates(githubHost, githubOwner, localRepo, remoteName)
			viper.Set("vault.kmskeyid.kms-pushed", true)
			viper.WriteConfig()
		}

		progressPrinter.IncrementTracker("step-github", 1)

		informUser("Attempt to recycle certs", globalFlags.SilentMode)
		restoreSSLCmd.RunE(cmd, args)
		progressPrinter.IncrementTracker("step-base", 1)

		gitopsRepo := fmt.Sprintf("git@github.com:%s/gitops.git", viper.GetString("github.owner"))

		botPrivateKey := viper.GetString("botprivatekey")

		argoCDConfig := argocd.GetArgoCDInitialCloudConfig(gitopsRepo, botPrivateKey)

		err = argocd.CreateInitialArgoCDRepository(config, argoCDConfig)
		if err != nil {
			return err
		}

		err = helm.InstallArgocd(globalFlags.DryRun)
		if err != nil {
			log.Println("Error installing argocd")
			return err
		}

		informUser("Install ArgoCD", globalFlags.SilentMode)
		progressPrinter.IncrementTracker("step-apps", 1)

		//! argocd was just helm installed
		waitArgoCDToBeReady(globalFlags.DryRun)
		informUser("ArgoCD Ready", globalFlags.SilentMode)

		kPortForwardArgocd, err = k8s.PortForward(globalFlags.DryRun, "argocd", "svc/argocd-server", "8080:80")
		defer kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
		informUser(fmt.Sprintf("ArgoCD available at %s", viper.GetString("argocd.local.service")), globalFlags.SilentMode)

		informUser("Setting argocd credentials", globalFlags.SilentMode)
		setArgocdCreds(globalFlags.DryRun)
		informUser("Getting an argocd auth token", globalFlags.SilentMode)
		token := argocd.GetArgocdAuthToken(globalFlags.DryRun)
		argocd.ApplyRegistry(globalFlags.DryRun)
		informUser("Syncing the registry application", globalFlags.SilentMode)
		informUser("Setup ArgoCD", globalFlags.SilentMode)
		progressPrinter.IncrementTracker("step-apps", 1)

		if globalFlags.DryRun {
			log.Printf("[#99] Dry-run mode, Sync ArgoCD skipped")
		} else {
			// todo: create ArgoCD struct, and host dependencies (like http client)
			customTransport := http.DefaultTransport.(*http.Transport).Clone()
			customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			httpClient := http.Client{Transport: customTransport}

			// retry to sync ArgoCD application until reaches the maximum attempts
			argoCDIsReady, err := argocd.SyncRetry(&httpClient, 20, 5, "registry", token)
			if err != nil {
				log.Printf("something went wrong during ArgoCD sync step, error is: %v", err)
			}

			if !argoCDIsReady {
				log.Println("unable to sync ArgoCD application, continuing...")
			}
		}
		informUser("Setup ArgoCD", globalFlags.SilentMode)
		progressPrinter.IncrementTracker("step-apps", 1)

		informUser("Waiting vault to be ready", globalFlags.SilentMode)
		waitVaultToBeRunning(globalFlags.DryRun)
		kPortForwardVault, err := k8s.PortForward(globalFlags.DryRun, "vault", "svc/vault", "8200:8200")
		defer kPortForwardVault.Process.Signal(syscall.SIGTERM)

		loopUntilPodIsReady(globalFlags.DryRun)
		initializeVaultAndAutoUnseal(globalFlags.DryRun)
		informUser(fmt.Sprintf("Vault available at %s", viper.GetString("vault.local.service")), globalFlags.SilentMode)
		informUser("Setup Vault", globalFlags.SilentMode)
		progressPrinter.IncrementTracker("step-apps", 1)

		if !viper.GetBool("vault.configuredsecret") { //skipVault
			informUser("waiting for vault unseal", globalFlags.SilentMode)
			log.Println("configuring vault")
			vault.ConfigureVault(globalFlags.DryRun)
			informUser("Vault configured", globalFlags.SilentMode)

			vault.GetOidcClientCredentials(globalFlags.DryRun)
			log.Println("vault oidc clients created")

			log.Println("creating vault configured secret")
			k8s.CreateVaultConfiguredSecret(globalFlags.DryRun, config)
			informUser("Vault secret created", globalFlags.SilentMode)
		}
		informUser("Terraform Vault", globalFlags.SilentMode)
		progressPrinter.IncrementTracker("step-apps", 1)

		// todo move this into a new command `kubefirst testDns --host argocd` ?
		// argocdPodClient := clientset.CoreV1().Pods("argocd")
		// for i := 1; i < 15; i++ {
		// 	argoCDHostReady := gitlab.AwaitHostNTimes("argocd", globalFlags.DryRun, 20)
		// 	if argoCDHostReady {
		// 		informUser("ArgoCD DNS is ready", globalFlags.SilentMode)
		// 		break
		// 	} else {
		// 		k8s.DeletePodByLabel(argocdPodClient, "app.kubernetes.io/name=argocd-server")
		// 	}
		// }

		directory = fmt.Sprintf("%s/gitops/terraform/users", config.K1FolderPath)
		informUser("applying users terraform", globalFlags.SilentMode)
		gitProvider := viper.GetString("git.mode")
		err = terraform.ApplyUsersTerraform(globalFlags.DryRun, directory, gitProvider)
		if err != nil {
			return err
		}
		progressPrinter.IncrementTracker("step-base", 1)
		//TODO: Do we need this?
		//From changes on create --> We need to fix once OIDC is ready
		if false {
			progressPrinter.AddTracker("step-vault-be", "Configure Vault Backend", 1)
			log.Println("configuring vault backend")
			vault.ConfigureVault(globalFlags.DryRun)
			informUser("Vault backend configured", globalFlags.SilentMode)
			progressPrinter.IncrementTracker("step-vault-be", 1)
		}
		progressPrinter.IncrementTracker("step-apps", 1)
		return nil
	},
}

func init() {
	clusterCmd.AddCommand(createGithubCmd)
	currentCommand := createGithubCmd
	flagset.DefineGithubCmdFlags(currentCommand)
	flagset.DefineGlobalFlags(currentCommand)
	// todo: make this an optional switch and check for it or viper
	currentCommand.Flags().Bool("skip-gitlab", false, "Skip GitLab lab install and vault setup")
	currentCommand.Flags().Bool("skip-vault", false, "Skip post-gitClient lab install and vault setup")
}
