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
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/helm"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/reports"
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

		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			log.Panic(err)
		}

		useTelemetry, err := cmd.Flags().GetBool("use-telemetry")
		if err != nil {
			log.Panic(err)
		}

		infoCmd.Run(cmd, args)
		progressPrinter.IncrementTracker("step-0", 1)

		progressPrinter.AddTracker("step-telemetry", "Send Telemetry", 4)
		sendStartedInstallTelemetry(dryRun, useTelemetry)
		progressPrinter.IncrementTracker("step-telemetry", 1)
		if !useTelemetry {
			informUser("Telemetry Disabled")
		}

		informUser("Creating gitops/metaphor repos")
		err = githubAddCmd.RunE(cmd, args)
		if err != nil {
			return err
		}

		informUser("populating gitops/metaphor repos")
		err = githubPopulateCmd.RunE(cmd, args)
		if err != nil {
			return err
		}

		directory := fmt.Sprintf("%s/gitops/terraform/base", config.K1FolderPath)
		informUser("Creating K8S Cluster")
		terraform.ApplyBaseTerraform(dryRun, directory)

		//progressPrinter.IncrementTracker("step-terraform", 1)

		//informUser("Attempt to recycle certs")
		//restoreSSLCmd.Run(cmd, args)

		/*

			progressPrinter.AddTracker("step-argo", "Deploy CI/CD ", 5)
			informUser("Deploy ArgoCD")
			progressPrinter.IncrementTracker("step-argo", 1)
		*/
		argocd.CreateInitalArgoRepository("git@github.com:kxdroid/gitops.git")

		clientset, err := k8s.GetClientSet()
		if err != nil {
			log.Printf("Failed to get clientset for k8s : %s", err)
			return err
		}
		helm.InstallArgocd(dryRun)

		//! argocd was just helm installed
		waitArgoCDToBeReady(dryRun)
		informUser("ArgoCD Ready")
		//progressPrinter.IncrementTracker("step-argo", 1)

		kPortForwardArgocd, err = k8s.K8sPortForward(dryRun, "argocd", "svc/argocd-server", "8080:80")
		defer kPortForwardArgocd.Process.Signal(syscall.SIGTERM)

		// log.Println("sleeping for 45 seconds, hurry up jared")
		// time.Sleep(45 * time.Second)

		informUser(fmt.Sprintf("ArgoCD available at %s", viper.GetString("argocd.local.service")))
		//progressPrinter.IncrementTracker("step-argo", 1)

		informUser("Setting argocd credentials")
		setArgocdCreds(dryRun)
		//progressPrinter.IncrementTracker("step-argo", 1)

		informUser("Getting an argocd auth token")
		token := argocd.GetArgocdAuthToken(dryRun)
		//progressPrinter.IncrementTracker("step-argo", 1)

		argocd.ApplyRegistry(dryRun)

		informUser("Syncing the registry application")

		if dryRun {
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

		return nil

		//!- Cesar Stops here
		// todo, need to stall until the registry has synced, then get to ui asap

		//! skip this if syncing from argocd and not helm installing
		// log.Printf("sleeping for 30 seconds, hurry up jared sign into argocd %s", viper.GetString("argocd.admin.password"))
		// time.Sleep(30 * time.Second)

		//!
		//* we need to stop here and wait for the vault namespace to exist and the vault pod to be ready
		//!
		progressPrinter.AddTracker("step-github", "Setup GitHub", 6)
		informUser("Waiting vault to be ready")
		waitVaultToBeRunning(dryRun)
		progressPrinter.IncrementTracker("step-github", 1)
		kPortForwardVault, err := k8s.K8sPortForward(dryRun, "vault", "svc/vault", "8200:8200")
		defer kPortForwardVault.Process.Signal(syscall.SIGTERM)

		loopUntilPodIsReady(dryRun)
		initializeVaultAndAutoUnseal(dryRun)
		informUser(fmt.Sprintf("Vault available at %s", viper.GetString("vault.local.service")))
		progressPrinter.IncrementTracker("step-github", 1)

		if !true { //skipVault

			progressPrinter.AddTracker("step-vault", "Configure Vault", 2)
			informUser("waiting for vault unseal")

			log.Println("configuring vault")
			vault.ConfigureVault(dryRun)
			informUser("Vault configured")
			progressPrinter.IncrementTracker("step-vault", 1)

			log.Println("creating vault configured secret")
			createVaultConfiguredSecret(dryRun, config)
			informUser("Vault  secret created")
			progressPrinter.IncrementTracker("step-vault", 1)
		}

		//gitlab oidc removed

		//i am here!

		if !viper.GetBool("github.gitops-pushed") {
			gitlab.PushGitRepo(dryRun, config, "gitlab", "gitops") // todo: need to handle if this was already pushed, errors on failure)
			progressPrinter.IncrementTracker("step-post-gitlab", 1)
			// todo: keep one of the two git push functions, they're similar, but not exactly the same
			//gitlab.PushGitOpsToGitLab(dryRun)
			viper.Set("gitlab.gitops-pushed", true)
			viper.WriteConfig()
		}
		if !dryRun && !viper.GetBool("argocd.oidc-patched") {
			argocdSecretClient = clientset.CoreV1().Secrets("argocd")
			patchSecret(argocdSecretClient, "argocd-secret", "oidc.gitlab.clientSecret", viper.GetString("gitlab.oidc.argocd.secret"))

			argocdPodClient := clientset.CoreV1().Pods("argocd")
			k8s.DeletePodByLabel(argocdPodClient, "app.kubernetes.io/name=argocd-server")
			viper.Set("argocd.oidc-patched", true)
			viper.WriteConfig()
		}
		if !viper.GetBool("gitlab.metaphor-pushed") {
			informUser("Pushing metaphor repo to origin gitlab")
			gitlab.PushGitRepo(dryRun, config, "gitlab", "metaphor")
			progressPrinter.IncrementTracker("step-post-gitlab", 1)
			// todo: keep one of the two git push functions, they're similar, but not exactly the same
			//gitlab.PushGitOpsToGitLab(dryRun)
			viper.Set("gitlab.metaphor-pushed", true)
			viper.WriteConfig()
		}
		if !viper.GetBool("gitlab.registered") {
			// informUser("Getting ArgoCD auth token")
			// token := argocd.GetArgocdAuthToken(dryRun)
			// progressPrinter.IncrementTracker("step-post-gitlab", 1)

			// informUser("Detaching the registry application from softserve")
			// argocd.DeleteArgocdApplicationNoCascade(dryRun, "registry", token)
			// progressPrinter.IncrementTracker("step-post-gitlab", 1)

			informUser("Adding the registry application registered against gitlab")
			gitlab.ChangeRegistryToGitLab(dryRun)
			progressPrinter.IncrementTracker("step-post-gitlab", 1)
			// todo triage / force apply the contents adjusting
			// todo kind: Application .repoURL:

			// informUser("Waiting for argocd host to resolve")
			// gitlab.AwaitHost("argocd", dryRun)
			if !dryRun {
				argocdPodClient := clientset.CoreV1().Pods("argocd")
				kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
				informUser("deleting argocd-server pod")
				k8s.DeletePodByLabel(argocdPodClient, "app.kubernetes.io/name=argocd-server")
			}
			informUser("waiting for argocd to be ready")
			waitArgoCDToBeReady(dryRun)

			informUser("Port forwarding to new argocd-server pod")
			time.Sleep(time.Second * 20)
			kPortForwardArgocd, err = k8s.K8sPortForward(dryRun, "argocd", "svc/argocd-server", "8080:80")
			defer kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
			log.Println("sleeping for 40 seconds")
			time.Sleep(40 * time.Second)

			informUser("Syncing the registry application")
			token := argocd.GetArgocdAuthToken(dryRun)

			if dryRun {
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

			viper.Set("gitlab.registered", true)
			viper.WriteConfig()
		}

		//!--
		// Wait argocd cert to work, or force restart
		argocdPodClient := clientset.CoreV1().Pods("argocd")
		for i := 1; i < 15; i++ {
			argoCDHostReady := gitlab.AwaitHostNTimes("argocd", dryRun, 20)
			if argoCDHostReady {
				informUser("ArgoCD DNS is ready")
				break
			} else {
				k8s.DeletePodByLabel(argocdPodClient, "app.kubernetes.io/name=argocd-server")
			}
		}

		//!--

		sendCompleteInstallTelemetry(dryRun, useTelemetry)
		time.Sleep(time.Millisecond * 100)

		// prepare data for the handoff report
		clusterData := reports.CreateHandOff{
			AwsAccountId:      viper.GetString("aws.accountid"),
			AwsHostedZoneName: viper.GetString("aws.hostedzonename"),
			AwsRegion:         viper.GetString("aws.region"),
			ClusterName:       viper.GetString("cluster-name"),

			GitlabURL:      fmt.Sprintf("https://gitlab.%s", viper.GetString("aws.hostedzonename")),
			GitlabUser:     "root",
			GitlabPassword: viper.GetString("gitlab.root.password"),

			RepoGitops:   fmt.Sprintf("https://gitlab.%s/kubefirst/gitops", viper.GetString("aws.hostedzonename")),
			RepoMetaphor: fmt.Sprintf("https://gitlab.%s/kubefirst/metaphor", viper.GetString("aws.hostedzonename")),

			VaultUrl:   fmt.Sprintf("https://vault.%s", viper.GetString("aws.hostedzonename")),
			VaultToken: viper.GetString("vault.token"),

			ArgoCDUrl:      fmt.Sprintf("https://argocd.%s", viper.GetString("aws.hostedzonename")),
			ArgoCDUsername: viper.GetString("argocd.admin.username"),
			ArgoCDPassword: viper.GetString("argocd.admin.password"),

			ArgoWorkflowsUrl: fmt.Sprintf("https://argo.%s", viper.GetString("aws.hostedzonename")),
			AtlantisUrl:      fmt.Sprintf("https://atlantis.%s", viper.GetString("aws.hostedzonename")),
			ChartMuseumUrl:   fmt.Sprintf("https://chartmuseum.%s", viper.GetString("aws.hostedzonename")),

			MetaphorDevUrl:        fmt.Sprintf("https://metaphor-development.%s", viper.GetString("aws.hostedzonename")),
			MetaphorStageUrl:      fmt.Sprintf("https://metaphor-staging.%s", viper.GetString("aws.hostedzonename")),
			MetaphorProductionUrl: fmt.Sprintf("https://metaphor-production.%s", viper.GetString("aws.hostedzonename")),
		}

		// build the string that will be sent to the report
		handOffData := reports.BuildCreateHandOffReport(clusterData)
		// call handoff report and apply style
		reports.CommandSummary(handOffData)

		fmt.Println("createGithub called")
		progressPrinter.GetInstance()
		progressPrinter.SetupProgress(4)
		//config := configs.ReadConfig()
		infoCmd.Run(cmd, args)

		progressPrinter.AddTracker("step-0", "Test Installer ", 4)
		//sendStartedInstallTelemetry(dryRun, useTelemetry)
		informUser("Create Github Org")
		informUser("Create Github Repo - gitops")
		//gitWrapper.CreatePrivateRepo("org-demo-6za", "gitops-template-foo", "My Foo Repo")
		//gitlab.PushGitRepo(dryRun, config, "gitlab", "metaphor")
		// make a github version of it

		//gitlab.PushGitRepo(dryRun, config, "gitlab", "metaphor")
		// make a github version of it

		informUser("Created Github Repo - gitops/metaphor")

		//populate

		//gitlab.PushGitRepo(dryRun, config, "gitlab", "gitops")
		// make a github version of it
		informUser("Creating K8S Cluster")
		//terraform.ApplyBaseTerraform(dryRun, directory)

		// this should be handled by the process detokinize
		//!-New
		informUser("Point registry to github") // this should be handled by the process detokinize
		informUser("Add github runner")

		//!-Old
		informUser("Setup ArgoCD")
		informUser("Wait Vailt to be ready")
		informUser("Unseal Vault")
		informUser("Do we need terraform Github?")
		informUser("Setup Vault")
		informUser("Setup OICD - Github/Argo")
		informUser("Final Argo Synch")
		informUser("Wait ArgoCD to be ready")
		//sendCompleteInstallTelemetry(dryRun, useTelemetry)

		//!-New
		informUser("Show Hand-off screen")
		//reports.CreateHandOff
		//reports.CommandSummary(handOffData)
		time.Sleep(time.Millisecond * 2000)
		return nil
	},
}

func init() {
	clusterCmd.AddCommand(createGithubCmd)
	currentCommand := createGithubCmd
	currentCommand.Flags().String("github-org", "", "Github Org of repos")
	currentCommand.Flags().String("github-owner", "", "Github Owner of repos")
	currentCommand.Flags().String("github-host", "github.com", "Github repo, usally github.com, but it can change on enterprise customers.")
	// todo: make this an optional switch and check for it or viper
	currentCommand.Flags().Bool("destroy", false, "destroy resources")
	currentCommand.Flags().Bool("dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
	currentCommand.Flags().Bool("skip-gitlab", false, "Skip GitLab lab install and vault setup")
	currentCommand.Flags().Bool("skip-vault", false, "Skip post-gitClient lab install and vault setup")
	currentCommand.Flags().Bool("use-telemetry", true, "installer will not send telemetry about this installation")

}
