package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/downloadManager"
	"github.com/kubefirst/kubefirst/internal/telemetry"	
	"github.com/kubefirst/kubefirst/pkg"
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
		trackers[pkg.DownloadDependencies] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.DownloadDependencies, 3)}
		trackers[pkg.GetAccountInfo] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.GetAccountInfo, 1)}
		trackers[pkg.GetDNSInfo] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.GetDNSInfo, 1)}
		trackers[pkg.TestHostedZoneLiveness] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.TestHostedZoneLiveness, 1)}
		trackers[pkg.CloneAndDetokenizeGitOpsTemplate] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.CloneAndDetokenizeGitOpsTemplate, 1)}
		trackers[pkg.CloneAndDetokenizeMetaphorTemplate] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.CloneAndDetokenizeMetaphorTemplate, 1)}
		trackers[pkg.CreateSSHKey] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.CreateSSHKey, 1)}
		trackers[pkg.CreateBuckets] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.CreateBuckets, 1)}
		trackers[pkg.SendTelemetry] = &pkg.ActionTracker{Tracker: pkg.CreateTracker(pkg.SendTelemetry, 1)}
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

		// cluster name
		clusterName, err := cmd.Flags().GetString("cluster-name")
		if err != nil {
			log.Panic(err)
		}
		viper.Set("cluster-name", clusterName)
		log.Println("cluster-name:", clusterName)

		// version-gitops
		versionGitOps, err := cmd.Flags().GetString("version-gitops")
		if err != nil {
			log.Panic(err)
		}
		viper.Set("version-gitops", versionGitOps)
		log.Println("version-gitops:", versionGitOps)

		viper.WriteConfig()

		//! tracker 0
		log.Println("installing kubefirst dependencies")
		trackers[pkg.DownloadDependencies].Tracker.Increment(1)
		err = downloadManager.DownloadTools(config, trackers)
		if err != nil {
			log.Panic(err)
		}
		log.Println("dependency installation complete")
		trackers[pkg.DownloadDependencies].Tracker.Increment(1)

		//! tracker 1
		log.Println("getting aws account information")
		aws.GetAccountInfo()
		log.Printf("aws account id: %s\naws user arn: %s", viper.GetString("aws.accountid"), viper.GetString("aws.userarn"))
		trackers[pkg.GetAccountInfo].Tracker.Increment(1)

		//! tracker 2
		// hosted zone id
		// So we don't have to keep looking it up from the domain name to use it
		hostedZoneId := aws.GetDNSInfo(hostedZoneName)
		// viper values set in above function
		log.Println("hostedZoneId:", hostedZoneId)
		trackers[pkg.GetDNSInfo].Tracker.Increment(1)

		//! tracker 3
		// todo: this doesn't default to testing the dns check
		skipHostedZoneCheck := viper.GetBool("init.hostedzonecheck.enabled")
		if !skipHostedZoneCheck {
			log.Println("skipping hosted zone check")
		} else {
			aws.TestHostedZoneLiveness(dryRun, hostedZoneName, hostedZoneId)
		}
		trackers[pkg.TestHostedZoneLiveness].Tracker.Increment(1)

		//! tracker 4
		// todo: remove it after successful dry-run test
		//log.Println("calling cloneGitOpsRepo()")
		//gitClient.CloneGitOpsRepo()
		//log.Println("cloneGitOpsRepo() complete")
		// refactor: start
		// TODO: get the below line added as a legit flag, don't merge with any value except kubefirst
		gitopsTemplateGithubOrgOverride := "kubefirst" // <-- discussion point
		log.Printf("cloning and detokenizing the gitops-template repository")
		if gitopsTemplateGithubOrgOverride != "" {
			log.Printf("using --gitops-template-gh-org=%s", gitopsTemplateGithubOrgOverride)
		}

		//! tracker 5
		prepareKubefirstTemplateRepo(config, gitopsTemplateGithubOrgOverride, "gitops",viper.GetString("version-gitops"))
		log.Println("clone and detokenization of gitops-template repository complete")
		trackers[pkg.CloneAndDetokenizeGitOpsTemplate].Tracker.Increment(int64(1))
		//! tracker 6
		log.Printf("cloning and detokenizing the metaphor-template repository")
		prepareKubefirstTemplateRepo(config, "kubefirst", "metaphor","main")
		log.Println("clone and detokenization of metaphor-template repository complete")
		trackers[pkg.CloneAndDetokenizeMetaphorTemplate].Tracker.Increment(int64(1))

		//! tracker 7
		log.Println("creating an ssh key pair for your new cloud infrastructure")
		pkg.CreateSshKeyPair()
		log.Println("ssh key pair creation complete")
		trackers[pkg.CreateSSHKey].Tracker.Increment(1)

		//! tracker 8
		//* should we consider going down to a single bucket
		//* for state and artifacts on open source?
		//* hitting a bucket limit on an install might deter someone
		log.Println("creating buckets for state and artifacts")
		aws.BucketRand(dryRun, trackers)
		trackers[pkg.CreateBuckets].Tracker.Increment(1)
		log.Println("BucketRand() complete")

		//! tracker 9
		log.Println("calling Detokenize()")
		pkg.Detokenize(fmt.Sprintf("%s/.kubefirst/gitops", config.HomePath))
		log.Println("Detokenize() complete")

		metricName = "kubefirst.init.completed"

		if !dryRun {
			telemetry.SendTelemetry(metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}

		viper.WriteConfig()

		//! tracker 10
		trackers[pkg.SendTelemetry].Tracker.Increment(1)
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
