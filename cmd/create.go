package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/kubefirst/nebulous/internal/telemetry"
	"github.com/kubefirst/nebulous/pkg/flare"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const trackerStage20 = "0 - Apply Base"
const trackerStage21 = "1 - Temporary SCM Install"
const trackerStage22 = "2 - Argo/Final SCM Install"
const trackerStage23 = "3 - Final Setup"

var skipVault bool
var skipGitlab bool

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		flare.SetupProgress(4)
		Trackers = make(map[string]*flare.ActionTracker)
		Trackers[trackerStage20] = &flare.ActionTracker{Tracker: flare.CreateTracker(trackerStage20, int64(1))}
		Trackers[trackerStage21] = &flare.ActionTracker{Tracker: flare.CreateTracker(trackerStage21, int64(2))}
		Trackers[trackerStage22] = &flare.ActionTracker{Tracker: flare.CreateTracker(trackerStage22, int64(7))}
		Trackers[trackerStage23] = &flare.ActionTracker{Tracker: flare.CreateTracker(trackerStage23, int64(3))}

		infoCmd.Run(cmd, args)

		metricName := "kubefirst.mgmt_cluster_install.started"
		metricDomain := viper.GetString("aws.hostedzonename")

		if !dryrunMode {
			telemetry.SendTelemetry(metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}

		directory := fmt.Sprintf("%s/.kubefirst/gitops/terraform/base", home)
		applyBaseTerraform(cmd, directory)
		Trackers[trackerStage20].Tracker.Increment(int64(1))

		//! soft-serve was just applied

		createSoftServe()
		waitForNamespaceandPods("soft-serve", "app=soft-serve")
		// todo this should be replaced with something more intelligent
		log.Println("waiting for soft-serve installation to complete...")
		time.Sleep(60 * time.Second)

		kPortForwardSoftServe := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "soft-serve", "port-forward", "svc/soft-serve", "8022:22")
		kPortForwardSoftServe.Stdout = os.Stdout
		kPortForwardSoftServe.Stderr = os.Stderr
		err := kPortForwardSoftServe.Start()
		defer kPortForwardSoftServe.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Panicf("error: failed to port-forward to soft-serve %s", err)
		}
		time.Sleep(20 * time.Second)

		configureSoftserveAndPush()
		Trackers[trackerStage21].Tracker.Increment(int64(1))
		helmInstallArgocd(home, kubeconfigPath)
		Trackers[trackerStage22].Tracker.Increment(int64(1))

		//! argocd was just helm installed
		x := 50
		for i := 0; i < x; i++ {
			kGetNamespace := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "get", "namespace/argocd")
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
			kGetNamespace := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "get", "pods", "-l", "app.kubernetes.io/name=argocd-server")
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

		kPortForwardArgocd := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "argocd", "port-forward", "svc/argocd-server", "8080:80")
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
		token := getArgocdAuthToken()
		log.Println("syncing the registry application")
		syncArgocdApplication("registry", token)
		// todo, need to stall until the registry has synced, then get to ui asap

		//! skip this if syncing from argocd and not helm installing
		log.Printf("sleeping for 30 seconds, hurry up jared sign into argocd %s", viper.GetString("argocd.admin.password"))
		time.Sleep(30 * time.Second)

		//!
		//* we need to stop here and wait for the vault namespace to exist and the vault pod to be ready
		//!
		x = 50
		for i := 0; i < x; i++ {
			kGetNamespace := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "get", "namespace/vault")
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
			kGetNamespace := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "vault", "get", "pods", "-l", "vault-initialized=true")
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
		kPortForwardVault := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "vault", "port-forward", "svc/vault", "8200:8200")
		kPortForwardVault.Stdout = os.Stdout
		kPortForwardVault.Stderr = os.Stderr
		err = kPortForwardVault.Start()
		defer kPortForwardVault.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Panicf("error: failed to port-forward to vault in main thread %s", err)
		}

		x = 50
		for i := 0; i < x; i++ {
			kGetNamespace := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "get", "namespace/gitlab")
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
			kGetNamespace := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "gitlab", "get", "pods", "-l", "app=webservice")
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
		waitForGitlab()
		log.Println("gitlab is ready!")
		kPortForwardGitlab := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "gitlab", "port-forward", "svc/gitlab-webservice-default", "8888:8080")
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
			produceGitlabTokens()
			Trackers[trackerStage22].Tracker.Increment(int64(1))
			applyGitlabTerraform(directory)
			Trackers[trackerStage22].Tracker.Increment(int64(1))
			gitlabKeyUpload()
			Trackers[trackerStage22].Tracker.Increment(int64(1))
		}
		if !skipVault {

			log.Println("waiting for vault unseal")
			waitForVaultUnseal()
			log.Println("vault unseal condition met - continuing")

			log.Println("configuring vault")
			configureVault()
			log.Println("vault configured")

			log.Println("creating vault configured secret")
			createVaultConfiguredSecret()
			log.Println("vault-configured secret created")

			Trackers[trackerStage23].Tracker.Increment(int64(1))
			addGitlabOidcApplications()
			Trackers[trackerStage23].Tracker.Increment(int64(1))

			log.Println("waiting for gitlab dns to propogate before continuing")
			awaitGitlab()

			log.Println("pushing gitops repo to origin gitlab")
			// pushGitRepo("gitlab", "gitops") //  todo need to handle if this was already pushed, errors on failure
			Trackers[trackerStage22].Tracker.Increment(int64(1))

			log.Println("pushing metaphor repo to origin gitlab")
			pushGitRepo("gitlab", "metaphor")
			Trackers[trackerStage23].Tracker.Increment(int64(1))

			changeRegistryToGitLab()
			Trackers[trackerStage22].Tracker.Increment(int64(1))

			// todo triage / force apply the contents adjusting
			// todo kind: Application .repoURL:

			// token := viper.GetString("argocd.admin.apitoken")
			// syncArgocdApplication("argo-components", token)
			// syncArgocdApplication("gitlab-runner-components", token)
			// syncArgocdApplication("gitlab-runner", token)
			// syncArgocdApplication("atlantis-components", token)
			// syncArgocdApplication("chartmuseum-components", token)

		}

		metricName = "kubefirst.mgmt_cluster_install.completed"

		if !dryrunMode {
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
	createCmd.PersistentFlags().BoolVarP(&dryrunMode, "dry-run", "s", false, "set to dry-run mode, no changes done on cloud provider selected")
	createCmd.PersistentFlags().BoolVar(&skipVault, "skip-vault", false, "Skip post-git lab install and vault setup")
	createCmd.PersistentFlags().BoolVar(&skipGitlab, "skip-gitlab", false, "Skip git lab install and vault setup")
}

func setArgocdCreds() {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
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
