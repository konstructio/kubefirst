// /*
// Copyright © 2022 NAME HERE <EMAIL ADDRESS>
// */
package cluster

import (
	"crypto/tls"
	"fmt"
	"github.com/kubefirst/kubefirst/cmd/cli/tools"
	"github.com/kubefirst/kubefirst/pkg"
	"log"
	"net/http"
	"os/exec"
	"syscall"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/helm"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/vault"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func CreateGitHubCommand() *cobra.Command {

	createGitHubCmd := &cobra.Command{
		Use:   "create-github",
		Short: "create a kubefirst management cluster with github as Git Repo",
		Long:  `Create a kubefirst cluster using github as the Git Repo and setup integrations`,
		RunE:  runCreateGitHubCommand,
	}

	return createGitHubCmd
}

func runCreateGitHubCommand(cmd *cobra.Command, args []string) error {

	config := configs.ReadConfig()
	//globalFlags, err := flagset.ProcessGlobalFlags(cmd)
	//if err != nil {
	//	return err
	//}

	// todo: this is temporary, command flags should be independent, and has no dependency from other commands
	silentMode, err := cmd.Flags().GetBool("silent")
	if err != nil {
		log.Println(err)
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		log.Println(err)
	}

	progressPrinter.SetupProgress(4, silentMode)

	//infoCmd need to be before the bars or it is printed in between bars:
	//Let's try to not move it on refactors
	//info.infoCmd.Run(cmd, args)
	tools.RunInfo(cmd, args)
	var kPortForwardArgocd *exec.Cmd
	progressPrinter.AddTracker("step-0", "Process Parameters", 1)
	progressPrinter.AddTracker("step-github", "Setup gitops on github", 3)
	progressPrinter.AddTracker("step-base", "Setup base cluster", 3)
	progressPrinter.AddTracker("step-apps", "Install apps to cluster", 6)

	progressPrinter.IncrementTracker("step-0", 1)

	//if !globalFlags.UseTelemetry {
	//	pkg.InformUser("Telemetry Disabled", silentMode)
	//}

	//* create github teams in the org and gitops repo
	pkg.InformUser("Creating gitops/metaphor repos", silentMode)
	err = githubAddCmd.RunE(cmd, args)
	if err != nil {
		log.Println("Error running:", githubAddCmd.Name())
		return err
	}

	pkg.InformUser(fmt.Sprintf("Created GitOps Repo in github.com/%s", viper.GetString("github.owner")), silentMode)
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
	pkg.InformUser("Creating K8S Cluster", silentMode)
	terraform.ApplyBaseTerraform(dryRun, directory)
	progressPrinter.IncrementTracker("step-base", 1)

	// pushes detokenized KMS_KEY_ID
	if !viper.GetBool("vault.kms.kms-pushed") {
		gitClient.PushLocalRepoUpdates(githubHost, githubOwner, localRepo, remoteName)
		viper.Set("vault.kmskeyid.kms-pushed", true)
		viper.WriteConfig()
	}

	progressPrinter.IncrementTracker("step-github", 1)

	pkg.InformUser("Attempt to recycle certs", silentMode)
	restoreSSLCmd.RunE(cmd, args)
	progressPrinter.IncrementTracker("step-base", 1)

	gitopsRepo := fmt.Sprintf("git@github.com:%s/gitops.git", viper.GetString("github.owner"))
	argocd.CreateInitialArgoCDRepository(gitopsRepo)

	// clientset, err := k8s.GetClientSet(dryRun)
	// if err != nil {
	// 	log.Printf("Failed to get clientset for k8s : %s", err)
	// 	return err
	// }
	err = helm.InstallArgocd(dryRun)
	if err != nil {
		log.Println("Error installing argocd")
		return err
	}

	pkg.InformUser("Install ArgoCD", silentMode)
	progressPrinter.IncrementTracker("step-apps", 1)

	//! argocd was just helm installed
	waitArgoCDToBeReady(dryRun)
	pkg.InformUser("ArgoCD Ready", silentMode)

	kPortForwardArgocd, err = k8s.PortForward(dryRun, "argocd", "svc/argocd-server", "8080:80")
	defer kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
	pkg.InformUser(fmt.Sprintf("ArgoCD available at %s", viper.GetString("argocd.local.service")), silentMode)

	pkg.InformUser("Setting argocd credentials", silentMode)
	setArgocdCreds(dryRun)
	pkg.InformUser("Getting an argocd auth token", silentMode)
	token := argocd.GetArgocdAuthToken(dryRun)
	argocd.ApplyRegistry(dryRun)
	pkg.InformUser("Syncing the registry application", silentMode)
	pkg.InformUser("Setup ArgoCD", silentMode)
	progressPrinter.IncrementTracker("step-apps", 1)

	if dryRun {
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
	pkg.InformUser("Setup ArgoCD", silentMode)
	progressPrinter.IncrementTracker("step-apps", 1)

	pkg.InformUser("Waiting vault to be ready", silentMode)
	waitVaultToBeRunning(dryRun)
	kPortForwardVault, err := k8s.PortForward(dryRun, "vault", "svc/vault", "8200:8200")
	defer kPortForwardVault.Process.Signal(syscall.SIGTERM)

	loopUntilPodIsReady(dryRun)
	initializeVaultAndAutoUnseal(dryRun)
	pkg.InformUser(fmt.Sprintf("Vault available at %s", viper.GetString("vault.local.service")), silentMode)
	pkg.InformUser("Setup Vault", silentMode)
	progressPrinter.IncrementTracker("step-apps", 1)

	if !viper.GetBool("vault.configuredsecret") { //skipVault
		pkg.InformUser("waiting for vault unseal", silentMode)
		log.Println("configuring vault")
		vault.ConfigureVault(dryRun)
		pkg.InformUser("Vault configured", silentMode)

		vault.GetOidcClientCredentials(dryRun)
		log.Println("vault oidc clients created")

		log.Println("creating vault configured secret")
		k8s.CreateVaultConfiguredSecret(dryRun, config)
		pkg.InformUser("Vault secret created", silentMode)
	}
	pkg.InformUser("Terraform Vault", silentMode)
	progressPrinter.IncrementTracker("step-apps", 1)

	// todo move this into a new command `kubefirst testDns --host argocd` ?
	// argocdPodClient := clientset.CoreV1().Pods("argocd")
	// for i := 1; i < 15; i++ {
	// 	argoCDHostReady := gitlab.AwaitHostNTimes("argocd", dryRun, 20)
	// 	if argoCDHostReady {
	// 		pkg.InformUser("ArgoCD DNS is ready", silentMode)
	// 		break
	// 	} else {
	// 		k8s.DeletePodByLabel(argocdPodClient, "app.kubernetes.io/name=argocd-server")
	// 	}
	// }

	directory = fmt.Sprintf("%s/gitops/terraform/users", config.K1FolderPath)
	pkg.InformUser("applying users terraform", silentMode)
	gitProvider := viper.GetString("git.mode")
	err = terraform.ApplyUsersTerraform(dryRun, directory, gitProvider)
	if err != nil {
		return err
	}
	progressPrinter.IncrementTracker("step-base", 1)
	//TODO: Do we need this?
	//From changes on create --> We need to fix once OIDC is ready
	if false {
		progressPrinter.AddTracker("step-vault-be", "Configure Vault Backend", 1)
		log.Println("configuring vault backend")
		vault.ConfigureVault(dryRun)
		pkg.InformUser("Vault backend configured", silentMode)
		progressPrinter.IncrementTracker("step-vault-be", 1)
	}
	progressPrinter.IncrementTracker("step-apps", 1)
	return nil
}

func initialization() {
	//cmd.clusterCmd.AddCommand(createGithubCmd)
	//currentCommand := createGithubCmd
	//flagset.DefineGithubCmdFlags(currentCommand)
	//flagset.DefineGlobalFlags(currentCommand)
	//// todo: make this an optional switch and check for it or viper
	//currentCommand.Flags().Bool("skip-gitlab", false, "Skip GitLab lab install and vault setup")
	//currentCommand.Flags().Bool("skip-vault", false, "Skip post-gitClient lab install and vault setup")
}
