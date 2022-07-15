package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"syscall"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/helm"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/softserve"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/vault"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const trackerStage20 = "0 - Apply Base"
const trackerStage21 = "1 - Temporary SCM Install"
const trackerStage22 = "2 - Argo/Final SCM Install"
const trackerStage23 = "3 - Final Setup"

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "create a kubefirst management cluster",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		progressPrinter.AddTracker("step-0", "Process Parameters", 1)
		config := configs.ReadConfig()

		skipVault, err := cmd.Flags().GetBool("skip-vault")
		if err != nil {
			log.Panic(err)
		}
		skipGitlab, err := cmd.Flags().GetBool("skip-gitlab")
		if err != nil {
			log.Panic(err)
		}
		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			log.Panic(err)
		}

		infoCmd.Run(cmd, args)
		progressPrinter.IncrementTracker("step-0", 1)

		progressPrinter.AddTracker("step-softserve", "Prepare Temporary Repo ", 4)
		sendStartedInstallTelemetry(dryRun)
		progressPrinter.IncrementTracker("step-softserve", 1)

		directory := fmt.Sprintf("%s/gitops/terraform/base", config.K1FolderPath)
		informUser("Creating K8S Cluster")
		terraform.ApplyBaseTerraform(dryRun, directory)
		progressPrinter.IncrementTracker("step-softserve", 1)

		//! soft-serve was just applied

		softserve.CreateSoftServe(dryRun, config.KubeConfigPath)
		informUser("Created Softserve")
		progressPrinter.IncrementTracker("step-softserve", 1)
		informUser("Waiting Softserve")
		waitForNamespaceandPods(dryRun, config, "soft-serve", "app=soft-serve")
		progressPrinter.IncrementTracker("step-softserve", 1)
		// todo this should be replaced with something more intelligent
		log.Println("Waiting for soft-serve installation to complete...")
		if !dryRun {
			var kPortForwardSoftServeOutb, kPortForwardSoftServeErrb bytes.Buffer
			time.Sleep(60 * time.Second)
			kPortForwardSoftServe := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "soft-serve", "port-forward", "svc/soft-serve", "8022:22")
			kPortForwardSoftServe.Stdout = &kPortForwardSoftServeOutb
			kPortForwardSoftServe.Stderr = &kPortForwardSoftServeErrb
			err = kPortForwardSoftServe.Start()
			defer kPortForwardSoftServe.Process.Signal(syscall.SIGTERM)
			if err != nil {
				// If it doesn't error, we kinda don't care much.
				log.Println("Commad Execution STDOUT: %s", kPortForwardSoftServeOutb.String())
				log.Println("Commad Execution STDERR: %s", kPortForwardSoftServeErrb.String())
				log.Panicf("error: failed to port-forward to soft-serve %s", err)
			}
			time.Sleep(20 * time.Second)
		}

		informUser("Softserve Update")
		softserve.ConfigureSoftServeAndPush(dryRun)
		progressPrinter.IncrementTracker("step-softserve", 1)

		progressPrinter.AddTracker("step-argo", "Deploy CI/CD ", 5)
		informUser("Deploy ArgoCD")
		progressPrinter.IncrementTracker("step-argo", 1)
		helm.InstallArgocd(dryRun)

		//! argocd was just helm installed
		waitArgoCDToBeReady(dryRun)
		informUser("ArgoCD Ready")
		progressPrinter.IncrementTracker("step-argo", 1)
		if !dryRun {
			var kPortForwardArgocdOutb, kPortForwardArgocdErrb bytes.Buffer
			kPortForwardArgocd := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "port-forward", "svc/argocd-server", "8080:80")
			kPortForwardArgocd.Stdout = &kPortForwardArgocdOutb
			kPortForwardArgocd.Stderr = &kPortForwardArgocdErrb
			err = kPortForwardArgocd.Start()
			defer kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
			if err != nil {
				log.Println("Commad Execution STDOUT: %s", kPortForwardArgocdOutb.String())
				log.Println("Commad Execution STDERR: %s", kPortForwardArgocdErrb.String())
				log.Panicf("error: failed to port-forward to argocd in main thread %s", err)
			}

			log.Println("sleeping for 45 seconds, hurry up jared")
			time.Sleep(45 * time.Second)
		}
		informUser(fmt.Sprintf("ArgoCD available at %s", viper.GetString("argocd.local.service")))
		progressPrinter.IncrementTracker("step-argo", 1)

		informUser("Setting argocd credentials")
		setArgocdCreds(dryRun)
		progressPrinter.IncrementTracker("step-argo", 1)

		informUser("Getting an argocd auth token")
		token := argocd.GetArgocdAuthToken(dryRun)
		progressPrinter.IncrementTracker("step-argo", 1)

		informUser("Syncing the registry application")
		argocd.SyncArgocdApplication(dryRun, "registry", token)
		progressPrinter.IncrementTracker("step-argo", 1)

		// todo, need to stall until the registry has synced, then get to ui asap

		//! skip this if syncing from argocd and not helm installing
		log.Printf("sleeping for 30 seconds, hurry up jared sign into argocd %s", viper.GetString("argocd.admin.password"))
		time.Sleep(30 * time.Second)

		//!
		//* we need to stop here and wait for the vault namespace to exist and the vault pod to be ready
		//!
		progressPrinter.AddTracker("step-gitlab", "Setup Gitlab", 6)
		informUser("Waiting vault to be ready")
		waitVaultToBeRunning(dryRun)
		progressPrinter.IncrementTracker("step-gitlab", 1)
		if !dryRun {
			var kPortForwardVaultOutb, kPortForwardVaultErrb bytes.Buffer
			kPortForwardVault := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "vault", "port-forward", "svc/vault", "8200:8200")
			kPortForwardVault.Stdout = &kPortForwardVaultOutb
			kPortForwardVault.Stderr = &kPortForwardVaultErrb
			err = kPortForwardVault.Start()
			defer kPortForwardVault.Process.Signal(syscall.SIGTERM)
			if err != nil {
				// If it doesn't error, we kinda don't care much.
				log.Printf("Commad Execution STDOUT: %s", kPortForwardVaultOutb.String())
				log.Printf("Commad Execution STDERR: %s", kPortForwardVaultErrb.String())
				log.Panicf("error: failed to port-forward to vault in main thread %s", err)
			}
		}
		loopUntilPodIsReady()
		initializeVaultAndAutoUnseal()
		informUser(fmt.Sprintf("Vault available at %s", viper.GetString("vault.local.service")))
		progressPrinter.IncrementTracker("step-gitlab", 1)

		informUser("Waiting gitlab to be ready")
		waitGitlabToBeReady(dryRun)
		log.Println("waiting for gitlab")
		waitForGitlab(dryRun, config)
		log.Println("gitlab is ready!")
		progressPrinter.IncrementTracker("step-gitlab", 1)

		if !dryRun {
			var kPortForwardGitlabOutb, kPortForwardGitlabErrb bytes.Buffer
			kPortForwardGitlab := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "gitlab", "port-forward", "svc/gitlab-webservice-default", "8888:8080")
			kPortForwardGitlab.Stdout = &kPortForwardGitlabOutb
			kPortForwardGitlab.Stderr = &kPortForwardGitlabErrb
			err = kPortForwardGitlab.Start()
			defer kPortForwardGitlab.Process.Signal(syscall.SIGTERM)
			if err != nil {
				// If it doesn't error, we kinda don't care much.
				log.Println("Commad Execution STDOUT: %s", kPortForwardGitlabOutb.String())
				log.Println("Commad Execution STDERR: %s", kPortForwardGitlabErrb.String())
				log.Panicf("error: failed to port-forward to gitlab in main thread %s", err)
			}
		}
		informUser(fmt.Sprintf("Gitlab available at %s", viper.GetString("gitlab.local.service")))
		progressPrinter.IncrementTracker("step-gitlab", 1)

		if !skipGitlab {
			// TODO: Confirm if we need to waitgit lab to be ready
			// OR something, too fast the secret will not be there.
			informUser("Gitlab setup tokens")
			gitlab.ProduceGitlabTokens(dryRun)
			progressPrinter.IncrementTracker("step-gitlab", 1)
			informUser("Gitlab terraform")
			gitlab.ApplyGitlabTerraform(dryRun, directory)
			gitlab.GitlabKeyUpload(dryRun)
			informUser("Gitlab ready")
			progressPrinter.IncrementTracker("step-gitlab", 1)

			if !skipVault {

				progressPrinter.AddTracker("step-vault", "Configure Vault", 4)
				informUser("waiting for vault unseal")
				/**

				 */
				waitVaultToBeRunning(dryRun)
				informUser("Vault running")
				progressPrinter.IncrementTracker("step-vault", 1)

				waitForVaultUnseal(dryRun, config)
				informUser("Vault unseal")
				progressPrinter.IncrementTracker("step-vault", 1)

				log.Println("configuring vault")
				vault.ConfigureVault(dryRun)
				informUser("Vault configured")
				progressPrinter.IncrementTracker("step-vault", 1)

				log.Println("creating vault configured secret")
				createVaultConfiguredSecret(dryRun, config)
				informUser("Vault  secret created")
				progressPrinter.IncrementTracker("step-vault", 1)
			}

			if !viper.GetBool("gitlab.oidc-created") {
				progressPrinter.AddTracker("step-post-gitlab", "Finalize Gitlab updates", 5)
				vault.AddGitlabOidcApplications(dryRun)
				informUser("Added Gitlab OIDC")
				progressPrinter.IncrementTracker("step-post-gitlab", 1)

				informUser("Waiting for Gitlab dns to propagate before continuing")
				gitlab.AwaitGitlab(dryRun)
				progressPrinter.IncrementTracker("step-post-gitlab", 1)

				informUser("Pushing gitops repo to origin gitlab")
				// refactor: sounds like a new functions, should PushGitOpsToGitLab be renamed/update signature?
				viper.Set("gitlab.oidc-created", true)
				viper.WriteConfig()
			}
			if !viper.GetBool("gitlab.gitops-pushed") {
				gitlab.PushGitRepo(dryRun, config, "gitlab", "gitops") // todo: need to handle if this was already pushed, errors on failure)
				progressPrinter.IncrementTracker("step-post-gitlab", 1)
				// todo: keep one of the two git push functions, they're similar, but not exactly the same
				//gitlab.PushGitOpsToGitLab(dryRun)
				viper.Set("gitlab.gitops-pushed", true)
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
				informUser("Changing registry to Gitlab")
				gitlab.ChangeRegistryToGitLab(dryRun)
				progressPrinter.IncrementTracker("step-post-gitlab", 1)
				// todo triage / force apply the contents adjusting
				// todo kind: Application .repoURL:
				viper.Set("gitlab.registered", true)
				viper.WriteConfig()
			}
		}
		sendCompleteInstallTelemetry(dryRun)
		time.Sleep(time.Millisecond * 100)

		if dryRun {
			log.Println("no handoff data on dry-run mode")
			return
		}

		// prepare data for the handoff report
		clusterData := reports.CreateHandOff{
			ClusterName:       viper.GetString("cluster-name"),
			AwsAccountId:      viper.GetString("aws.accountid"),
			AwsHostedZoneName: viper.GetString("aws.hostedzonename"),
			AwsRegion:         viper.GetString("aws.region"),
			ArgoCDUrl:         viper.GetString("argocd.local.service"),
			ArgoCDUsername:    viper.GetString("argocd.admin.username"),
			ArgoCDPassword:    viper.GetString("argocd.admin.password"),
			VaultUrl:          viper.GetString("vault.local.service"),
			VaultToken:        viper.GetString("vault.token"),
		}

		// build the string that will be sent to the report
		handOffData := reports.BuildCreateHandOffReport(clusterData)

		// call handoff report and apply style
		reports.CommandSummary(handOffData)

	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	// todo: make this an optional switch and check for it or viper
	createCmd.Flags().Bool("destroy", false, "destroy resources")
	createCmd.Flags().Bool("dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
	createCmd.Flags().Bool("skip-gitlab", false, "Skip GitLab lab install and vault setup")
	createCmd.Flags().Bool("skip-vault", false, "Skip post-gitClient lab install and vault setup")

	progressPrinter.GetInstance()
	progressPrinter.SetupProgress(4)
}
