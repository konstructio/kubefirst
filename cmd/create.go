package cmd

import (
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/helm"
	"github.com/kubefirst/kubefirst/internal/softserve"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/vault"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
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

		pkg.SetupProgress(4)
		Trackers := make(map[string]*pkg.ActionTracker)

		Trackers[trackerStage20] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(trackerStage20, 1)}
		Trackers[trackerStage21] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(trackerStage21, 2)}
		Trackers[trackerStage22] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(trackerStage22, 7)}
		Trackers[trackerStage23] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(trackerStage23, 3)}

		infoCmd.Run(cmd, args)

		
		sendStartedInstallTelemetry(dryRun)


		directory := fmt.Sprintf("%s/gitops/terraform/base", config.K1srtFolderPath)
		terraform.ApplyBaseTerraform(dryRun, directory)
		Trackers[trackerStage20].Tracker.Increment(int64(1))

		//! soft-serve was just applied

		softserve.CreateSoftServe(dryRun, config.KubeConfigPath)
		waitForNamespaceandPods(config, "soft-serve", "app=soft-serve")
		// todo this should be replaced with something more intelligent
		log.Println("waiting for soft-serve installation to complete...")
		time.Sleep(60 * time.Second)

		kPortForwardSoftServe := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "soft-serve", "port-forward", "svc/soft-serve", "8022:22")
		kPortForwardSoftServe.Stdout = os.Stdout
		kPortForwardSoftServe.Stderr = os.Stderr
		err = kPortForwardSoftServe.Start()
		defer kPortForwardSoftServe.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Panicf("error: failed to port-forward to soft-serve %s", err)
		}
		time.Sleep(20 * time.Second)

		Trackers[trackerStage21].Tracker.Increment(int64(1))
		softserve.ConfigureSoftServeAndPush(dryRun)
		Trackers[trackerStage21].Tracker.Increment(int64(1))
		helm.InstallArgocd(dryRun)
		Trackers[trackerStage22].Tracker.Increment(int64(1))

		//! argocd was just helm installed
		waitArgoCDToBeReady()

		kPortForwardArgocd := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "port-forward", "svc/argocd-server", "8080:80")
		kPortForwardArgocd.Stdout = os.Stdout
		kPortForwardArgocd.Stderr = os.Stderr
		err = kPortForwardArgocd.Start()
		defer kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Panicf("error: failed to port-forward to argocd in main thread %s", err)
		}

		log.Println("sleeping for 45 seconds, hurry up jared")
		time.Sleep(45 * time.Second)

		log.Println("setting argocd credentials")
		setArgocdCreds()
		log.Println("getting an argocd auth token")
		token := argocd.GetArgocdAuthToken(dryRun)
		log.Println("syncing the registry application")
		argocd.SyncArgocdApplication(dryRun, "registry", token)
		// todo, need to stall until the registry has synced, then get to ui asap

		//! skip this if syncing from argocd and not helm installing
		log.Printf("sleeping for 30 seconds, hurry up jared sign into argocd %s", viper.GetString("argocd.admin.password"))
		time.Sleep(30 * time.Second)

		//!
		//* we need to stop here and wait for the vault namespace to exist and the vault pod to be ready
		//!
		waitVaultToBeInitialized()		
		kPortForwardVault := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "vault", "port-forward", "svc/vault", "8200:8200")
		kPortForwardVault.Stdout = os.Stdout
		kPortForwardVault.Stderr = os.Stderr
		err = kPortForwardVault.Start()
		defer kPortForwardVault.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Panicf("error: failed to port-forward to vault in main thread %s", err)
		}
		waitGitlabToBeReady()
		log.Println("waiting for gitlab")
		waitForGitlab(config)
		log.Println("gitlab is ready!")
		kPortForwardGitlab := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "gitlab", "port-forward", "svc/gitlab-webservice-default", "8888:8080")
		kPortForwardGitlab.Stdout = os.Stdout
		kPortForwardGitlab.Stderr = os.Stderr
		err = kPortForwardGitlab.Start()
		defer kPortForwardGitlab.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Panicf("error: failed to port-forward to gitlab in main thread %s", err)
		}

		if !skipGitlab {
			// TODO: Confirm if we need to waitgit lab to be ready
			// OR something, too fast the secret will not be there.
			gitlab.ProduceGitlabTokens(dryRun)
			Trackers[trackerStage22].Tracker.Increment(int64(1))
			gitlab.ApplyGitlabTerraform(dryRun, directory)
			Trackers[trackerStage22].Tracker.Increment(int64(1))
			gitlab.GitlabKeyUpload(dryRun)
			Trackers[trackerStage22].Tracker.Increment(int64(1))

			if !skipVault {

				log.Println("waiting for vault unseal")
				/**

				 */
				waitVaultToBeInitialized()				
				waitForVaultUnseal(config)
				log.Println("vault unseal condition met - continuing")

				log.Println("configuring vault")
				vault.ConfigureVault(dryRun)
				log.Println("vault configured")

				log.Println("creating vault configured secret")
				createVaultConfiguredSecret(config)
				log.Println("vault-configured secret created")

				Trackers[trackerStage23].Tracker.Increment(int64(1))
				vault.AddGitlabOidcApplications(dryRun)
				Trackers[trackerStage23].Tracker.Increment(int64(1))

				log.Println("waiting for gitlab dns to propagate before continuing")
				gitlab.AwaitGitlab(dryRun)
				Trackers[trackerStage22].Tracker.Increment(int64(1))

				log.Println("pushing gitops repo to origin gitlab")
				// refactor: sounds like a new functions, should PushGitOpsToGitLab be renamed/update signature?

				gitlab.PushGitRepo(config, "gitlab", "gitops") // todo: need to handle if this was already pushed, errors on failure)
				// todo: keep one of the two git push functions, they're similar, but not exactly the same
				//gitlab.PushGitOpsToGitLab(dryRun)

				log.Println("pushing metaphor repo to origin gitlab")
				gitlab.PushGitRepo(config, "gitlab", "metaphor")
				// todo: keep one of the two git push functions, they're similar, but not exactly the same
				//gitlab.PushGitOpsToGitLab(dryRun)
				Trackers[trackerStage23].Tracker.Increment(int64(1))

				Trackers[trackerStage22].Tracker.Increment(int64(1))
				gitlab.ChangeRegistryToGitLab(dryRun)
				Trackers[trackerStage22].Tracker.Increment(int64(1))

				Trackers[trackerStage23].Tracker.Increment(int64(1))

				// todo triage / force apply the contents adjusting
				// todo kind: Application .repoURL:
			}
		}
		sendCompleteInstallTelemetry(dryRun)
		time.Sleep(time.Millisecond * 100)
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	// todo: make this an optional switch and check for it or viper
	createCmd.Flags().Bool("destroy", false, "destroy resources")
	createCmd.Flags().Bool("dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
	createCmd.Flags().Bool("skip-gitlab", false, "Skip GitLab lab install and vault setup")
	createCmd.Flags().Bool("skip-vault", false, "Skip post-gitClient lab install and vault setup")

}

