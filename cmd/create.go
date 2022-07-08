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
	"log"
	"time"
)

const trackerStage20 = "0 - Apply Base"
const trackerStage21 = "1 - Temporary SCM Install"
const trackerStage22 = "2 - Argo/Final SCM Install"
const trackerStage23 = "3 - Final Setup"

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
		metricDomain := viper.GetString("aws.domainname")

		if !dryRun {
			telemetry.SendTelemetry(metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}

		directory := fmt.Sprintf("%s/.kubefirst/gitops/terraform/base", config.HomePath)
		terraform.ApplyBaseTerraform(dryRun, directory)
		Trackers[trackerStage20].Tracker.Increment(int64(1))
		softserve.CreateSoftServe(dryRun, config.KubeConfigPath)
		Trackers[trackerStage21].Tracker.Increment(int64(1))
		softserve.ConfigureSoftServeAndPush(dryRun)
		Trackers[trackerStage21].Tracker.Increment(int64(1))
		helm.InstallArgocd(dryRun)
		Trackers[trackerStage22].Tracker.Increment(int64(1))

		if !skipGitlab {
			//TODO: Confirm if we need to waitgit lab to be ready
			// OR something, too fast the secret will not be there.
			gitlab.AwaitGitlab(dryRun)
			gitlab.ProduceGitlabTokens(dryRun)
			Trackers[trackerStage22].Tracker.Increment(int64(1))
			gitlab.ApplyGitlabTerraform(dryRun, directory)
			Trackers[trackerStage22].Tracker.Increment(int64(1))
			gitlab.GitlabKeyUpload(dryRun)
			Trackers[trackerStage22].Tracker.Increment(int64(1))

			if !skipVault {
				vault.ConfigureVault(dryRun)
				Trackers[trackerStage23].Tracker.Increment(int64(1))
				vault.AddGitlabOidcApplications(dryRun)
				Trackers[trackerStage23].Tracker.Increment(int64(1))
				gitlab.AwaitGitlab(dryRun)
				Trackers[trackerStage22].Tracker.Increment(int64(1))

				gitlab.PushGitOpsToGitLab(dryRun)
				Trackers[trackerStage22].Tracker.Increment(int64(1))
				gitlab.ChangeRegistryToGitLab(dryRun)
				Trackers[trackerStage22].Tracker.Increment(int64(1))

				gitlab.HydrateGitlabMetaphorRepo(dryRun)

				Trackers[trackerStage23].Tracker.Increment(int64(1))

				token := argocd.GetArgocdAuthToken(dryRun)
				argocd.SyncArgocdApplication(dryRun, "argo-components", token)
				argocd.SyncArgocdApplication(dryRun, "gitlab-runner-components", token)
				argocd.SyncArgocdApplication(dryRun, "gitlab-runner", token)
				argocd.SyncArgocdApplication(dryRun, "atlantis-components", token)
				argocd.SyncArgocdApplication(dryRun, "chartmuseum-components", token)
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
