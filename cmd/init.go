package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
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

		useTelemetry, err := cmd.Flags().GetBool("use-telemetry")
		if err != nil {
			log.Panic(err)
		}

		if !useTelemetry {
			log.Println("telemetry is disabled")
		}

		log.Println("dry run enabled:", dryRun)

		arnRole, err := cmd.Flags().GetString("aws-assume-role")
		if err != nil {
			log.Println("unable to use the provided AWS IAM role for AssumeRole feature")
			return
		}

		if len(arnRole) > 0 {
			log.Println("calling assume role")
			err := aws.AssumeRole(arnRole)
			if err != nil {
				log.Println(err)
				return
			}
			log.Printf("assuming new AWS credentials based on role %q", arnRole)
		}

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

		k1Dir := fmt.Sprintf("%s", config.K1FolderPath)
		if _, err := os.Stat(k1Dir); errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(k1Dir, os.ModePerm); err != nil {
				log.Panicf("info: could not create directory %q - error: %s", config.K1FolderPath, err)
			}
		} else {
			log.Printf("info: %s already exist", k1Dir)
		}

		infoCmd.Run(cmd, args)
		hostedZoneName, _ := cmd.Flags().GetString("hosted-zone-name")
		metricName := "kubefirst.init.started"
		metricDomain := hostedZoneName

		if !dryRun {
			telemetry.SendTelemetry(useTelemetry, metricDomain, metricName)
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

		// set region
		region, err := cmd.Flags().GetString("region")
		if err != nil {
			log.Panicf("unable to get region values from viper")
		}
		viper.Set("aws.region", region)
		// propagate it to local environment
		err = os.Setenv("AWS_REGION", region)
		if err != nil {
			log.Panicf("unable to set environment variable AWS_REGION, error is: %v", err)
		}
		log.Println("region:", region)

		// set profile
		profile, err := cmd.Flags().GetString("profile")
		if err != nil {
			log.Panicf("unable to get region values from viper")
		}
		viper.Set("aws.profile", profile)
		// propagate it to local environment
		err = os.Setenv("AWS_PROFILE", profile)
		if err != nil {
			log.Panicf("unable to set environment variable AWS_PROFILE, error is: %v", err)
		}
		log.Println("profile:", profile)

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

		// version-gitops
		templateTag, err := cmd.Flags().GetString("template-tag")
		if err != nil {
			log.Panic(err)
		}
		viper.Set("template.tag", templateTag)
		log.Println("template-tag:", templateTag)

		bucketRand, err := cmd.Flags().GetString("s3-suffix")
		if err != nil {
			log.Panic(err)
		}
		viper.Set("bucket.rand", bucketRand)
		log.Println("s3-suffix:", clusterName)

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
			aws.TestHostedZoneLiveness(dryRun, hostedZoneName, hostedZoneId)
		} else {
			log.Println("skipping hosted zone check")
		}
		trackers[pkg.TestHostedZoneLiveness].Tracker.Increment(1)

		//! tracker 4
		//* should we consider going down to a single bucket
		//* for state and artifacts on open source?
		//* hitting a bucket limit on an install might deter someone
		log.Println("creating buckets for state and artifacts")
		aws.BucketRand(dryRun)
		trackers[pkg.CreateBuckets].Tracker.Increment(1)
		log.Println("BucketRand() complete")

		//! tracker 5
		log.Println("creating an ssh key pair for your new cloud infrastructure")
		pkg.CreateSshKeyPair()
		log.Println("ssh key pair creation complete")
		trackers[pkg.CreateSSHKey].Tracker.Increment(1)

		//! tracker 6
		// TODO: get the below line added as a legit flag, don't merge with any value except kubefirst
		gitopsTemplateGithubOrgOverride := "kubefirst" // <-- discussion point
		log.Printf("cloning and detokenizing the gitops-template repository")
		if gitopsTemplateGithubOrgOverride != "" {
			log.Printf("using --gitops-template-gh-org=%s", gitopsTemplateGithubOrgOverride)
		}
		prepareKubefirstTemplateRepo(config, gitopsTemplateGithubOrgOverride, "gitops", viper.GetString("version-gitops"), viper.GetString("template.tag"))
		log.Println("clone and detokenization of gitops-template repository complete")
		trackers[pkg.CloneAndDetokenizeGitOpsTemplate].Tracker.Increment(int64(1))

		//! tracker 7
		log.Printf("cloning and detokenizing the metaphor-template repository")
		prepareKubefirstTemplateRepo(config, "kubefirst", "metaphor", "", viper.GetString("template.tag"))
		log.Println("clone and detokenization of metaphor-template repository complete")
		trackers[pkg.CloneAndDetokenizeMetaphorTemplate].Tracker.Increment(int64(1))

		metricName = "kubefirst.init.completed"

		if !dryRun {
			telemetry.SendTelemetry(useTelemetry, metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}

		viper.WriteConfig()

		//! tracker 8
		trackers[pkg.SendTelemetry].Tracker.Increment(1)
		time.Sleep(time.Millisecond * 100)
	},
}

func init() {
	config := configs.ReadConfig()

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
	initCmd.Flags().String("region", "eu-west-1", "the region to provision the cloud resources in")
	err = initCmd.MarkFlagRequired("region")
	if err != nil {
		log.Panic(err)
	}

	initCmd.Flags().String("profile", "default", "AWS profile located at ~/.aws/config")
	err = initCmd.MarkFlagRequired("profile")
	if err != nil {
		log.Panic(err)
	}
	initCmd.Flags().Bool("clean", false, "delete any local kubefirst content ~/.kubefirst, ~/.k1")

	log.SetPrefix("LOG: ")
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)

	initCmd.Flags().Bool("dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
	log.Println("init started")

	initCmd.Flags().String("cluster-name", "kubefirst", "the cluster name, used to identify resources on cloud provider")
	initCmd.Flags().String("s3-suffix", "", "unique identifier for s3 buckets")
	//We should try to synch this with newer naming
	initCmd.Flags().String("version-gitops", "", "version/branch used on git clone")
	initCmd.Flags().String("template-tag", config.KubefirstVersion, `fallback tag used on git clone.
Details: if "version-gitops" is provided, branch("version-gitops") has precedence and installer will attempt to clone branch("version-gitops") first,
if it fails, then fallback it will attempt to clone the tag provided at "template-tag" flag`)

	// AWS assume role
	initCmd.Flags().String("aws-assume-role", "", "instead of using AWS IAM user credentials, AWS AssumeRole feature generate role based credentials, more at https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html")
	initCmd.Flags().Bool("use-telemetry", true, "installer will not send telemetry about this installation")
}
