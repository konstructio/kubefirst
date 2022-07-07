package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

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
		metricDomain := viper.GetString("aws.domainname")

		if !dryrunMode {
			flare.SendTelemetry(metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}

		directory := fmt.Sprintf("%s/.kubefirst/gitops/terraform/base", home)
		applyBaseTerraform(cmd, directory)
		Trackers[trackerStage20].Tracker.Increment(int64(1))
		createSoftServe(kubeconfigPath)
		Trackers[trackerStage21].Tracker.Increment(int64(1))
		configureSoftserveAndPush()
		Trackers[trackerStage21].Tracker.Increment(int64(1))
		helmInstallArgocd(home, kubeconfigPath)
		Trackers[trackerStage22].Tracker.Increment(int64(1))

		log.Println("sleeping for 10 seconds, hurry up jared")
		time.Sleep(10 * time.Second)

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

		// todo do without this or make it meaningful
		log.Println("waiting for vault unseal")
		waitForVaultUnseal()
		log.Println("vault unseal condition met - continuing")

		log.Println("configuring vault")
		configureVault()
		log.Println("vault configured")

		var output bytes.Buffer
		// todo - https://github.com/bcreane/k8sutils/blob/master/utils.go
		// kubectl create secret generic vault-configured --from-literal=isConfigured=true
		// the purpose of this command is to let the vault-unseal Job running in kuberenetes know that external secrets store should be able to connect to the configured vault
		k := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "vault", "create", "secret", "generic", "vault-configured", "--from-literal=isConfigured=true")
		k.Stdout = &output
		k.Stderr = os.Stderr
		err := k.Run()
		if err != nil {
			log.Panicf("failed to create secret for vault-configured: %s", err)
		}
		log.Println("the output is: %s", output.String())

		return

		if !skipGitlab {
			//TODO: Confirm if we need to waitgit lab to be ready
			// OR something, too fast the secret will not be there.
			// awaitGitlab()
			// produceGitlabTokens()
			// Trackers[trackerStage22].Tracker.Increment(int64(1))
			// applyGitlabTerraform(directory)
			// Trackers[trackerStage22].Tracker.Increment(int64(1))
			// gitlabKeyUpload()
			// Trackers[trackerStage22].Tracker.Increment(int64(1))

			if !skipVault {
				Trackers[trackerStage23].Tracker.Increment(int64(1))
				addGitlabOidcApplications()
				Trackers[trackerStage23].Tracker.Increment(int64(1))
				awaitGitlab()
				Trackers[trackerStage22].Tracker.Increment(int64(1))

				pushGitopsToGitLab()
				Trackers[trackerStage22].Tracker.Increment(int64(1))
				changeRegistryToGitLab()
				Trackers[trackerStage22].Tracker.Increment(int64(1))

				hydrateGitlabMetaphorRepo()
				Trackers[trackerStage23].Tracker.Increment(int64(1))

				token := viper.GetString("argocd.admin.apitoken")
				syncArgocdApplication("argo-components", token)
				syncArgocdApplication("gitlab-runner-components", token)
				syncArgocdApplication("gitlab-runner", token)
				syncArgocdApplication("atlantis-components", token)
				syncArgocdApplication("chartmuseum-components", token)
			}
		}

		metricName = "kubefirst.mgmt_cluster_install.completed"

		if !dryrunMode {
			flare.SendTelemetry(metricDomain, metricName)
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
