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
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/repo"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		infoCmd.Run(cmd, args)
		config := configs.ReadConfig()

		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}

		installerFlags, err := flagset.ProcessInstallerGenericFlags(cmd)
		if err != nil {
			return err
		}

		awsFlags, err := flagset.ProcessAwsFlags(cmd)
		if err != nil {
			return err
		}

		githubFlags, err := flagset.ProcessGithubAddCmdFlags(cmd)
		if err != nil {
			return err
		}

		progressPrinter.GetInstance()
		progressPrinter.SetupProgress(10, globalFlags.SilentMode)

		informUser(
			"Silent mode enabled, most of the UI prints wont be showed. Please check the logs for more details.\n",
			globalFlags.SilentMode,
		)

		log.Println("github:", githubFlags.GithubHost)
		log.Println("dry run enabled:", globalFlags.DryRun)

		if len(awsFlags.AssumeRole) > 0 {
			log.Println("calling assume role")
			err := aws.AssumeRole(awsFlags.AssumeRole)
			if err != nil {
				log.Println(err)
				return err
			}
			log.Printf("assuming new AWS credentials based on role %q", awsFlags.AssumeRole)
		}

		progressPrinter.AddTracker("step-download", pkg.DownloadDependencies, 3)
		progressPrinter.AddTracker("step-account", pkg.GetAccountInfo, 1)
		progressPrinter.AddTracker("step-dns", pkg.GetDNSInfo, 1)
		progressPrinter.AddTracker("step-live", pkg.TestHostedZoneLiveness, 1)
		progressPrinter.AddTracker("step-gitops", pkg.CloneAndDetokenizeGitOpsTemplate, 1)
		progressPrinter.AddTracker("step-ssh", pkg.CreateSSHKey, 1)
		progressPrinter.AddTracker("step-buckets", pkg.CreateBuckets, 1)
		progressPrinter.AddTracker("step-telemetry", pkg.SendTelemetry, 1)

		k1Dir := fmt.Sprintf("%s", config.K1FolderPath)
		if _, err := os.Stat(k1Dir); errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(k1Dir, os.ModePerm); err != nil {
				log.Panicf("info: could not create directory %q - error: %s", config.K1FolderPath, err)
			}
		} else {
			log.Printf("info: %s already exist", k1Dir)
		}

		metricName := "kubefirst.init.started"
		metricDomain := awsFlags.HostedZoneName

		if !globalFlags.DryRun {
			telemetry.SendTelemetry(globalFlags.UseTelemetry, metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}

		// todo need to check flags and create config

		// hosted zone name:
		// name of the hosted zone to be used for the kubefirst install
		// if suffixed with a dot (eg. kubefirst.com.), the dot will be stripped
		if strings.HasSuffix(awsFlags.HostedZoneName, ".") {
			awsFlags.HostedZoneName = awsFlags.HostedZoneName[:len(awsFlags.HostedZoneName)-1]
		}
		log.Println("hostedZoneName:", awsFlags.HostedZoneName)

		viper.Set("argocd.local.service", "http://localhost:8080")
		viper.Set("gitlab.local.service", "http://localhost:8888")
		viper.Set("vault.local.service", "http://localhost:8200")
		// used for letsencrypt notifications and the gitlab root account

		log.Println("s3-suffix:", installerFlags.ClusterName)

		viper.WriteConfig()

		//! tracker 0
		log.Println("installing kubefirst dependencies")
		progressPrinter.IncrementTracker("step-download", 1)
		err = downloadManager.DownloadTools(config)
		if err != nil {
			return err
		}
		log.Println("dependency installation complete")
		progressPrinter.IncrementTracker("step-download", 1)
		//Fix incomplete bar, please don't remove it.
		progressPrinter.IncrementTracker("step-download", 1)

		//! tracker 1
		log.Println("getting aws account information")
		aws.GetAccountInfo()
		log.Printf("aws account id: %s\naws user arn: %s", viper.GetString("aws.accountid"), viper.GetString("aws.userarn"))
		progressPrinter.IncrementTracker("step-account", 1)

		//! tracker 2
		// hosted zone id
		// So we don't have to keep looking it up from the domain name to use it
		hostedZoneId := aws.GetDNSInfo(awsFlags.HostedZoneName)
		// viper values set in above function
		log.Println("hostedZoneId:", hostedZoneId)
		progressPrinter.IncrementTracker("step-dns", 1)

		//! tracker 3
		// todo: this doesn't default to testing the dns check
		skipHostedZoneCheck := viper.GetBool("init.hostedzonecheck.enabled")
		if !skipHostedZoneCheck {
			hostedZoneLiveness := aws.TestHostedZoneLiveness(globalFlags.DryRun, awsFlags.HostedZoneName, hostedZoneId)
			if !hostedZoneLiveness {
				log.Panic("Fail to check the Liveness of HostedZone, we need a valid public HostedZone on the same AWS account that Kubefirst will be installed.")
			}
		} else {
			log.Println("skipping hosted zone check")
		}
		progressPrinter.IncrementTracker("step-live", 1)

		//! tracker 4
		//* should we consider going down to a single bucket
		//* for state and artifacts on open source?
		//* hitting a bucket limit on an install might deter someone
		log.Println("creating buckets for state and artifacts")
		aws.BucketRand(globalFlags.DryRun)
		progressPrinter.IncrementTracker("step-buckets", 1)
		log.Println("BucketRand() complete")

		//! tracker 5
		log.Println("creating an ssh key pair for your new cloud infrastructure")
		pkg.CreateSshKeyPair()
		log.Println("ssh key pair creation complete")
		progressPrinter.IncrementTracker("step-ssh", 1)

		//! tracker 6
		repo.PrepareKubefirstTemplateRepo(globalFlags.DryRun, config, viper.GetString("gitops.owner"), viper.GetString("gitops.repo"), viper.GetString("gitops.branch"), viper.GetString("template.tag"))
		log.Println("clone and detokenization of gitops-template repository complete")
		progressPrinter.IncrementTracker("step-gitops", 1)

		metricName = "kubefirst.init.completed"

		if !globalFlags.DryRun {
			telemetry.SendTelemetry(globalFlags.UseTelemetry, metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}

		viper.WriteConfig()

		//! tracker 8
		progressPrinter.IncrementTracker("step-telemetry", 1)
		time.Sleep(time.Millisecond * 100)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	currentCommand := initCmd
	log.Println("kubefirst started")
	log.SetPrefix("LOG: ")
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)

	// Do we need this?
	initCmd.Flags().Bool("clean", false, "delete any local kubefirst content ~/.kubefirst, ~/.k1")

	//Group Flags
	flagset.DefineGlobalFlags(currentCommand)
	flagset.DefineGithubCmdFlags(currentCommand)
	flagset.DefineAWSFlags(currentCommand)
	flagset.DefineInstallerGenericFlags(currentCommand)

	currentCommand.MarkFlagRequired("admin-email")
	currentCommand.MarkFlagRequired("cloud")
	currentCommand.MarkFlagRequired("hosted-zone-name")
	currentCommand.MarkFlagRequired("region")
	currentCommand.MarkFlagRequired("profile")

}
