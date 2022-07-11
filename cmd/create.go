package cmd

import (
	"fmt"
	"github.com/kubefirst/nebulous/configs"
	"github.com/kubefirst/nebulous/internal/argocd"
	"github.com/kubefirst/nebulous/internal/gitlab"
	"github.com/kubefirst/nebulous/internal/helm"
	"github.com/kubefirst/nebulous/internal/softserve"
	"github.com/kubefirst/nebulous/internal/telemetry"
	"github.com/kubefirst/nebulous/internal/terraform"
	"github.com/kubefirst/nebulous/internal/vault"
	"github.com/kubefirst/nebulous/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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

		metricName := "kubefirst.mgmt_cluster_install.started"
		metricDomain := viper.GetString("aws.hostedzonename")

		if !dryRun {
			telemetry.SendTelemetry(metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}

		directory := fmt.Sprintf("%s/.kubefirst/gitops/terraform/base", config.HomePath)
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
		x := 50
		for i := 0; i < x; i++ {
			kGetNamespace := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "namespace/argocd")
			kGetNamespace.Stdout = os.Stdout
			kGetNamespace.Stderr = os.Stderr
			err := kGetNamespace.Run()
			if err != nil {
				log.Println("Waiting argocd to be born")
				time.Sleep(10 * time.Second)
			} else {
				log.Println("argocd namespace found, continuing")
				time.Sleep(5 * time.Second)
				break
			}
		}
		for i := 0; i < x; i++ {
			kGetNamespace := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "pods", "-l", "app.kubernetes.io/name=argocd-server")
			kGetNamespace.Stdout = os.Stdout
			kGetNamespace.Stderr = os.Stderr
			err := kGetNamespace.Run()
			if err != nil {
				log.Println("Waiting for argocd pods to create, checking in 10 seconds")
				time.Sleep(10 * time.Second)
			} else {
				log.Println("argocd pods found, continuing")
				time.Sleep(15 * time.Second)
				break
			}
		}

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
		x = 50
		for i := 0; i < x; i++ {
			kGetNamespace := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "namespace/vault")
			kGetNamespace.Stdout = os.Stdout
			kGetNamespace.Stderr = os.Stderr
			err := kGetNamespace.Run()
			if err != nil {
				log.Println("Waiting vault to be born")
				time.Sleep(10 * time.Second)
			} else {
				log.Println("vault namespace found, continuing")
				time.Sleep(25 * time.Second)
				break
			}
		}
		x = 50
		for i := 0; i < x; i++ {
			kGetNamespace := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "vault", "get", "pods", "-l", "vault-initialized=true")
			kGetNamespace.Stdout = os.Stdout
			kGetNamespace.Stderr = os.Stderr
			err := kGetNamespace.Run()
			if err != nil {
				log.Println("Waiting vault pods to create")
				time.Sleep(10 * time.Second)
			} else {
				log.Println("vault pods found, continuing")
				time.Sleep(15 * time.Second)
				break
			}
		}
		kPortForwardVault := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "vault", "port-forward", "svc/vault", "8200:8200")
		kPortForwardVault.Stdout = os.Stdout
		kPortForwardVault.Stderr = os.Stderr
		err = kPortForwardVault.Start()
		defer kPortForwardVault.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Panicf("error: failed to port-forward to vault in main thread %s", err)
		}

		x = 50
		for i := 0; i < x; i++ {
			kGetNamespace := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "namespace/gitlab")
			kGetNamespace.Stdout = os.Stdout
			kGetNamespace.Stderr = os.Stderr
			err := kGetNamespace.Run()
			if err != nil {
				log.Println("Waiting gitlab namespace to be born")
				time.Sleep(10 * time.Second)
			} else {
				log.Println("gitlab namespace found, continuing")
				time.Sleep(5 * time.Second)
				break
			}
		}
		x = 50
		for i := 0; i < x; i++ {
			kGetNamespace := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "gitlab", "get", "pods", "-l", "app=webservice")
			kGetNamespace.Stdout = os.Stdout
			kGetNamespace.Stderr = os.Stderr
			err := kGetNamespace.Run()
			if err != nil {
				log.Println("Waiting gitlab pods to be born")
				time.Sleep(10 * time.Second)
			} else {
				log.Println("gitlab pods found, continuing")
				time.Sleep(15 * time.Second)
				break
			}
		}
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
				gitlab.PushGitOpsToGitLab(dryRun)

				Trackers[trackerStage22].Tracker.Increment(int64(1))
				gitlab.ChangeRegistryToGitLab(dryRun)
				Trackers[trackerStage22].Tracker.Increment(int64(1))

				// refactor: should this be removed?
				gitlab.HydrateGitlabMetaphorRepo(dryRun)

				Trackers[trackerStage23].Tracker.Increment(int64(1))

				// todo triage / force apply the contents adjusting
				// todo kind: Application .repoURL:

				// refactor: should this be deleted?
				//token := argocd.GetArgocdAuthToken(dryRun)
				//argocd.SyncArgocdApplication(dryRun, "argo-components", token)
				//argocd.SyncArgocdApplication(dryRun, "gitlab-runner-components", token)
				//argocd.SyncArgocdApplication(dryRun, "gitlab-runner", token)
				//argocd.SyncArgocdApplication(dryRun, "atlantis-components", token)
				//argocd.SyncArgocdApplication(dryRun, "chartmuseum-components", token)
			}
		}

		metricName = "kubefirst.mgmt_cluster_install.completed"

		if !dryRun {
			telemetry.SendTelemetry(metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}
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

// todo: move it to internals/ArgoCD
func setArgocdCreds() {
	cfg := configs.ReadConfig()
	config, err := clientcmd.BuildConfigFromFlags("", cfg.KubeConfigPath)
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	argocdSecretClient = clientset.CoreV1().Secrets("argocd")

	argocdPassword := getSecretValue(argocdSecretClient, "argocd-initial-admin-secret", "password")

	viper.Set("argocd.admin.password", argocdPassword)
	viper.Set("argocd.admin.username", "admin")
	viper.WriteConfig()
}
