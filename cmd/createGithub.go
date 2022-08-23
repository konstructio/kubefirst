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
	"github.com/kubefirst/kubefirst/internal/gitlab"
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

		progressPrinter.GetInstance()
		progressPrinter.SetupProgress(4)

		var kPortForwardArgocd *exec.Cmd
		progressPrinter.AddTracker("step-0", "Process Parameters", 1)
		config := configs.ReadConfig()
		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}

		skipVault, err := cmd.Flags().GetBool("skip-vault")
		if err != nil {
			log.Panic(err)
		}

		infoCmd.Run(cmd, args)
		progressPrinter.IncrementTracker("step-0", 1)

		if !globalFlags.UseTelemetry {
			informUser("Telemetry Disabled")
		}
		informUser("Creating gitops/metaphor repos")
		err = githubAddCmd.RunE(cmd, args)
		if err != nil {
			log.Println("Error running:", githubAddCmd.Name())
			return err
		}
		err = loadTemplateCmd.RunE(cmd, args)
		if err != nil {
			log.Println("Error running loadTemplateCmd")
			return err
		}

		directory := fmt.Sprintf("%s/gitops/terraform/base", config.K1FolderPath)
		informUser("Creating K8S Cluster")
		terraform.ApplyBaseTerraform(globalFlags.DryRun, directory)

		informUser("populating gitops/metaphor repos")
		err = githubPopulateCmd.RunE(cmd, args)
		if err != nil {
			log.Println("Error running githubPopulateCmd")
			return err
		}

		//progressPrinter.IncrementTracker("step-terraform", 1)

		informUser("Attempt to recycle certs")
		restoreSSLCmd.Run(cmd, args)

		/*

			progressPrinter.AddTracker("step-argo", "Deploy CI/CD ", 5)
			informUser("Deploy ArgoCD")
			progressPrinter.IncrementTracker("step-argo", 1)
		*/
		gitopsRepo := fmt.Sprintf("git@github.com:%s/gitops.git", viper.GetString("github.owner"))
		argocd.CreateInitalArgoRepository(gitopsRepo)

		clientset, err := k8s.GetClientSet()
		if err != nil {
			log.Printf("Failed to get clientset for k8s : %s", err)
			return err
		}
		helm.InstallArgocd(globalFlags.DryRun)

		//! argocd was just helm installed
		waitArgoCDToBeReady(globalFlags.DryRun)
		informUser("ArgoCD Ready")
		//progressPrinter.IncrementTracker("step-argo", 1)

		kPortForwardArgocd, err = k8s.K8sPortForward(globalFlags.DryRun, "argocd", "svc/argocd-server", "8080:80")
		defer kPortForwardArgocd.Process.Signal(syscall.SIGTERM)

		// log.Println("sleeping for 45 seconds, hurry up jared")
		// time.Sleep(45 * time.Second)

		informUser(fmt.Sprintf("ArgoCD available at %s", viper.GetString("argocd.local.service")))
		//progressPrinter.IncrementTracker("step-argo", 1)

		informUser("Setting argocd credentials")
		setArgocdCreds(globalFlags.DryRun)
		//progressPrinter.IncrementTracker("step-argo", 1)

		informUser("Getting an argocd auth token")
		token := argocd.GetArgocdAuthToken(globalFlags.DryRun)
		//progressPrinter.IncrementTracker("step-argo", 1)

		argocd.ApplyRegistry(globalFlags.DryRun)

		informUser("Syncing the registry application")

		if globalFlags.DryRun {
			log.Printf("[#99] Dry-run mode, Sync ArgoCD skipped")
		} else {
			// todo: create ArgoCD struct, and host dependencies (like http client)
			customTransport := http.DefaultTransport.(*http.Transport).Clone()
			customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			httpClient := http.Client{Transport: customTransport}

			// retry to sync ArgoCD application until reaches the maximum attempts
			argoCDIsReady, err := argocd.SyncRetry(&httpClient, 60, 5, "registry", token)
			if err != nil {
				log.Printf("something went wrong during ArgoCD sync step, error is: %v", err)
			}

			if !argoCDIsReady {
				log.Println("unable to sync ArgoCD application, continuing...")
			}
		}

		//progressPrinter.IncrementTracker("step-argo", 1)
		//progressPrinter.AddTracker("step-github", "Setup GitHub", 6)
		informUser("Waiting vault to be ready")
		waitVaultToBeRunning(globalFlags.DryRun)
		//progressPrinter.IncrementTracker("step-github", 1)
		kPortForwardVault, err := k8s.K8sPortForward(globalFlags.DryRun, "vault", "svc/vault", "8200:8200")
		defer kPortForwardVault.Process.Signal(syscall.SIGTERM)

		loopUntilPodIsReady(globalFlags.DryRun)
		initializeVaultAndAutoUnseal(globalFlags.DryRun)
		informUser(fmt.Sprintf("Vault available at %s", viper.GetString("vault.local.service")))
		//progressPrinter.IncrementTracker("step-github", 1)

		if !skipVault { //skipVault

			//progressPrinter.AddTracker("step-vault", "Configure Vault", 2)
			informUser("waiting for vault unseal")

			log.Println("configuring vault")
			vault.ConfigureVault(globalFlags.DryRun, true)
			informUser("Vault configured")
			//progressPrinter.IncrementTracker("step-vault", 1)

			log.Println("creating vault configured secret")
			createVaultConfiguredSecret(globalFlags.DryRun, config)
			informUser("Vault  secret created")
			//progressPrinter.IncrementTracker("step-vault", 1)
		}

		//gitlab oidc removed

		argocdPodClient := clientset.CoreV1().Pods("argocd")
		for i := 1; i < 15; i++ {
			argoCDHostReady := gitlab.AwaitHostNTimes("argocd", globalFlags.DryRun, 20)
			if argoCDHostReady {
				informUser("ArgoCD DNS is ready")
				break
			} else {
				k8s.DeletePodByLabel(argocdPodClient, "app.kubernetes.io/name=argocd-server")
			}
		}

		//TODO: Do we need this?
		//From changes on create
		if !skipVault {
			progressPrinter.AddTracker("step-vault-be", "Configure Vault Backend", 1)
			log.Println("configuring vault backend")
			vault.ConfigureVault(globalFlags.DryRun, false)
			informUser("Vault backend configured")
			progressPrinter.IncrementTracker("step-vault-be", 1)
		}

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
