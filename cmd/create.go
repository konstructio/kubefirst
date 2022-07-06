package cmd

import (
	"fmt"
	"github.com/kubefirst/nebulous/pkg/flare"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"time"
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

		if !skipGitlab {
			//TODO: Confirm if we need to waitgit lab to be ready
			// OR something, too fast the secret will not be there.
			awaitGitlab()	
			produceGitlabTokens()
			Trackers[trackerStage22].Tracker.Increment(int64(1))				
			applyGitlabTerraform(directory)
			Trackers[trackerStage22].Tracker.Increment(int64(1))
			gitlabKeyUpload()
			Trackers[trackerStage22].Tracker.Increment(int64(1))
		
			if !skipVault {
				configureVault()
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
				
				token := getArgocdAuthToken()
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
