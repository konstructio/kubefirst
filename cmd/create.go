package cmd

import (
	"fmt"
	"github.com/kubefirst/nebulous/configs"
	"github.com/kubefirst/nebulous/internal/argocd"
	"github.com/kubefirst/nebulous/internal/gitlab"
	"github.com/kubefirst/nebulous/internal/helm"
	"github.com/kubefirst/nebulous/internal/telemetry"
	"github.com/kubefirst/nebulous/pkg"
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

		config := configs.ReadConfig()

		pkg.SetupProgress(4)
		Trackers := make(map[string]*pkg.ActionTracker)

		Trackers[trackerStage20] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(trackerStage20, 1)}
		Trackers[trackerStage21] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(trackerStage21, 2)}
		Trackers[trackerStage22] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(trackerStage22, 7)}
		Trackers[trackerStage23] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(trackerStage23, 3)}

		infoCmd.Run(cmd, args)

		metricName := "kubefirst.mgmt_cluster_install.started"
		metricDomain := viper.GetString("aws.domainname")

		if !config.DryRun {
			telemetry.SendTelemetry(metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}

		directory := fmt.Sprintf("%s/.kubefirst/gitops/terraform/base", config.HomePath)
		applyBaseTerraform(cmd, directory)
		Trackers[trackerStage20].Tracker.Increment(int64(1))
		createSoftServe(config.KubeConfigPath)
		Trackers[trackerStage21].Tracker.Increment(int64(1))
		configureSoftserveAndPush()
		Trackers[trackerStage21].Tracker.Increment(int64(1))
		helm.InstallArgocd(config.HomePath)
		Trackers[trackerStage22].Tracker.Increment(int64(1))

		if !skipGitlab {
			//TODO: Confirm if we need to waitgit lab to be ready
			// OR something, too fast the secret will not be there.
			gitlab.AwaitGitlab()
			gitlab.ProduceGitlabTokens()
			Trackers[trackerStage22].Tracker.Increment(int64(1))
			gitlab.ApplyGitlabTerraform(directory)
			Trackers[trackerStage22].Tracker.Increment(int64(1))
			gitlab.GitlabKeyUpload()
			Trackers[trackerStage22].Tracker.Increment(int64(1))

			if !skipVault {
				configureVault()
				Trackers[trackerStage23].Tracker.Increment(int64(1))
				addGitlabOidcApplications()
				Trackers[trackerStage23].Tracker.Increment(int64(1))
				gitlab.AwaitGitlab()
				Trackers[trackerStage22].Tracker.Increment(int64(1))

				gitlab.PushGitOpsToGitLab()
				Trackers[trackerStage22].Tracker.Increment(int64(1))
				gitlab.ChangeRegistryToGitLab()
				Trackers[trackerStage22].Tracker.Increment(int64(1))

				hydrateGitlabMetaphorRepo()
				Trackers[trackerStage23].Tracker.Increment(int64(1))

				token := argocd.GetArgocdAuthToken()
				argocd.SyncArgocdApplication("argo-components", token)
				argocd.SyncArgocdApplication("gitlab-runner-components", token)
				argocd.SyncArgocdApplication("gitlab-runner", token)
				argocd.SyncArgocdApplication("atlantis-components", token)
				argocd.SyncArgocdApplication("chartmuseum-components", token)
			}
		}

		metricName = "kubefirst.mgmt_cluster_install.completed"

		if !config.DryRun {
			telemetry.SendTelemetry(metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}
		time.Sleep(time.Millisecond * 100)
	},
}

func init() {
	config := configs.ReadConfig()
	rootCmd.AddCommand(createCmd)

	// todo: make this an optional switch and check for it or viper
	createCmd.Flags().Bool("destroy", false, "destroy resources")
	createCmd.PersistentFlags().BoolVarP(&config.DryRun, "dry-run", "s", false, "set to dry-run mode, no changes done on cloud provider selected")
	createCmd.PersistentFlags().BoolVar(&skipVault, "skip-vault", false, "Skip post-gitClient lab install and vault setup")
	createCmd.PersistentFlags().BoolVar(&skipGitlab, "skip-gitlab", false, "Skip gitClient lab install and vault setup")

}
