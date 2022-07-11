package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kubefirst/nebulous/configs"
	"github.com/kubefirst/nebulous/internal/aws"
	"github.com/kubefirst/nebulous/internal/downloadManager"
	"github.com/kubefirst/nebulous/internal/gitClient"
	"github.com/kubefirst/nebulous/internal/telemetry"
	"github.com/kubefirst/nebulous/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "initialize your local machine to execute `create`",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		config := configs.ReadConfig()

		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			log.Panic(err)
		}

		log.Println("dry run enabled:", dryRun)

		pkg.SetupProgress(10)
		trackers := pkg.GetTrackers()
		trackers[pkg.TrackerStage0] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.TrackerStage0, 1)}
		trackers[pkg.TrackerStage1] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.TrackerStage1, 1)}
		trackers[pkg.TrackerStage2] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.TrackerStage2, 1)}
		trackers[pkg.TrackerStage3] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.TrackerStage3, 1)}
		trackers[pkg.TrackerStage4] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.TrackerStage4, 1)}
		trackers[pkg.TrackerStage5] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.TrackerStage5, 3)}
		trackers[pkg.TrackerStage6] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.TrackerStage6, 1)}
		trackers[pkg.TrackerStage7] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.TrackerStage7, 3)}
		trackers[pkg.TrackerStage8] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.TrackerStage8, 1)}
		trackers[pkg.TrackerStage9] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.TrackerStage9, 1)}
		infoCmd.Run(cmd, args)
		hostedZoneName, _ := cmd.Flags().GetString("hosted-zone-name")
		metricName := "kubefirst.init.started"
		metricDomain := hostedZoneName

		if !dryRun {
			telemetry.SendTelemetry(metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}

		// todo need to check flags and create config

		// hosted zone name:
		// name of the hosted zone to be used for the kubefirst install
		// if suffixed with a dot (eg. kubefirst.com.), the dot will be stripped
		if strings.HasSuffix(hostedZoneName, ".") {
			hostedZoneName = hostedZoneName[:len(hostedZoneName)-1]
		}
		log.Println("hostedZoneName:", hostedZoneName)
		viper.Set("aws.hostedzonename", hostedZoneName)
		viper.Set("argocd.local.service", "http://localhost:8080")
		viper.Set("gitlab.local.service", "http://localhost:8888")
		viper.Set("vault.local.service", "http://localhost:8200")
		// admin email
		// used for letsencrypt notifications and the gitlab root account
		adminEmail, _ := cmd.Flags().GetString("admin-email")
		log.Println("adminEmail:", adminEmail)
		viper.Set("adminemail", adminEmail)

		// region
		// name of the cloud region to provision resources when resources are region-specific
		region, _ := cmd.Flags().GetString("region")
		viper.Set("aws.region", region)
		log.Println("region:", region)

		viper.WriteConfig()

		// refactor: confirm it (start)
		//! tracker 0
		log.Println("installing kubefirst dependencies")
		download()
		log.Println("dependency installation complete")
		Trackers[trackerStage0].Tracker.Increment(int64(1))

		//! tracker 1
		log.Println("getting aws account information")
		getAccountInfo()
		log.Printf("aws account id: %s\naws user arn: %s", viper.GetString("aws.accountid"), viper.GetString("aws.userarn"))
		Trackers[trackerStage1].Tracker.Increment(int64(1))
		// refactor: confirm it (end)

		// hosted zone id
		// so we don't have to keep looking it up from the domain name to use it
		hostedZoneId := aws.GetDNSInfo(hostedZoneName)
		// viper values set in above function
		log.Println("hostedZoneId:", hostedZoneId)
		trackers[pkg.TrackerStage0].Tracker.Increment(1)
		trackers[pkg.TrackerStage1].Tracker.Increment(1)

		//cluster name
		clusterName, err := cmd.Flags().GetString("cluster-name")
		if err != nil {
			log.Panic(err)
		}
		viper.Set("cluster-name", clusterName)
		log.Println("cluster-name:", clusterName)

		//version-gitops
		versionGitOps, err := cmd.Flags().GetString("version-gitops")
		if err != nil {
			log.Panic(err)
		}
		viper.Set("version-gitops", versionGitOps)
		log.Println("version-gitops:", versionGitOps)

		// todo: this doesn't default to testing the dns check
		skipHostedZoneCheck := viper.GetBool("init.hostedzonecheck.enabled")
		if !skipHostedZoneCheck {
			log.Println("skipping hosted zone check")
		} else {
			aws.TestHostedZoneLiveness(dryRun, hostedZoneName, hostedZoneId)
		}
		trackers[pkg.TrackerStage2].Tracker.Increment(1)

		log.Println("creating an ssh key pair for your new cloud infrastructure")
		pkg.CreateSshKeyPair()
		log.Println("ssh key pair creation complete")
		trackers[pkg.TrackerStage3].Tracker.Increment(1)

		log.Println("calling cloneGitOpsRepo()")
		gitClient.CloneGitOpsRepo()
		log.Println("cloneGitOpsRepo() complete")
		trackers[pkg.TrackerStage4].Tracker.Increment(1)

		log.Println("calling download()")
		trackers[pkg.TrackerStage5].Tracker.Increment(1)
		err = downloadManager.DownloadTools(config, trackers)
		if err != nil {
			log.Panic(err)
		}
		trackers[pkg.TrackerStage5].Tracker.Increment(1)

		log.Println("download() complete")

		log.Println("calling GetAccountInfo()")
		aws.GetAccountInfo()
		log.Println("GetAccountInfo() complete")
		trackers[pkg.TrackerStage6].Tracker.Increment(1)

		log.Println("calling BucketRand()")
		trackers[pkg.TrackerStage7].Tracker.Increment(1)

		//! tracker 4
		//* should we consider going down to a single bucket
		//* for state and artifacts on open source?
		//* hitting a bucket limit on an install might deter someone
		log.Println("creating buckets for state and artifacts")
		aws.BucketRand(dryRun, trackers)
		trackers[pkg.TrackerStage7].Tracker.Increment(1)
		log.Println("BucketRand() complete")

		log.Println("calling Detokenize()")
		pkg.Detokenize(fmt.Sprintf("%s/.kubefirst/gitops", config.HomePath))
		log.Println("Detokenize() complete")
		trackers[pkg.TrackerStage8].Tracker.Increment(1)

		// TODO: get the below line added as a legit flag, don't merge with any value except kubefirst
		gitopsTemplateGithubOrgOverride := "kubefirst" // <-- discussion point
		log.Printf("cloning and detokenizing the gitops-template repository")
		if gitopsTemplateGithubOrgOverride != "" {
			log.Printf("using --gitops-template-gh-org=%s", gitopsTemplateGithubOrgOverride)
		}

		//! tracker 6
		prepareKubefirstTemplateRepo(gitopsTemplateGithubOrgOverride, "gitops")
		log.Println("clone and detokenization of gitops-template repository complete")
		Trackers[trackerStage6].Tracker.Increment(int64(1))
		//! tracker 7
		log.Printf("cloning and detokenizing the metaphor-template repository")
		prepareKubefirstTemplateRepo("kubefirst", "metaphor")
		log.Println("clone and detokenization of metaphor-template repository complete")
		Trackers[trackerStage7].Tracker.Increment(int64(1))

		metricName = "kubefirst.init.completed"

		if !dryRun {
			telemetry.SendTelemetry(metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}

		viper.WriteConfig()

		//! tracker 8
		trackers[pkg.TrackerStage9].Tracker.Increment(1)
		time.Sleep(time.Millisecond * 100)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().String("hosted-zone-name", "", "the domain to provision the kubefirst platform in")
	err := initCmd.MarkFlagRequired("hosted-zone-name")
	if err != nil {
		log.Panic(err)
	}
	initCmd.Flags().String("admin-email", "", "the email address for the administrator as well as for lets-encrypt certificate emails")
	err = initCmd.MarkFlagRequired("admin-email")
	if err != nil {
		log.Panic(err)
	}
	initCmd.Flags().String("cloud", "", "the cloud to provision infrastructure in")
	err = initCmd.MarkFlagRequired("cloud")
	if err != nil {
		log.Panic(err)
	}
	initCmd.Flags().String("region", "", "the region to provision the cloud resources in")
	err = initCmd.MarkFlagRequired("region")
	if err != nil {
		log.Panic(err)
	}
	initCmd.Flags().Bool("clean", false, "delete any local kubefirst content ~/.kubefirst, ~/.flare")

	log.SetPrefix("LOG: ")
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)

	initCmd.Flags().Bool("dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
	log.Println("init started")

	initCmd.Flags().String("cluster-name", "k1st", "the cluster name, used to identify resources on cloud provider")
	initCmd.Flags().String("version-gitops", "main", "version/branch used on git clone")
}
