package cmd

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/segmentio/analytics-go"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/domain"
	"github.com/kubefirst/kubefirst/internal/downloadManager"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/repo"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize your local machine to execute `create`",
	Long: `Initialize the required resources to provision a full Cloud environment. At this step initial resources are
validated and configured.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		infoCmd.Run(cmd, args)
		config := configs.ReadConfig()

		//Please don't change the order of this block, wihtout updating
		// internal/flagset/init_test.go

		if err := pkg.ValidateK1Folder(config.K1FolderPath); err != nil {
			return err
		}

		// command line flags
		cloudValue, err := flagset.ReadConfigString(cmd, "cloud")
		if err != nil {
			return err
		}

		if cloudValue == flagset.CloudK3d {
			if config.GitHubPersonalAccessToken == "" {

				httpClient := http.DefaultClient
				gitHubService := services.NewGitHubService(httpClient)
				gitHubHandler := handlers.NewGitHubHandler(gitHubService)
				gitHubAccessToken, err := gitHubHandler.AuthenticateUser()
				if err != nil {
					return err
				}

				if len(gitHubAccessToken) == 0 {
					return errors.New("unable to retrieve a GitHub token for the user")
				}

				// todo: set common way to load env. values (viper->struct->load-env)
				if err := os.Setenv("KUBEFIRST_GITHUB_AUTH_TOKEN", gitHubAccessToken); err != nil {
					return err
				}
				log.Println("\nKUBEFIRST_GITHUB_AUTH_TOKEN set via OAuth")
			}
		}

		providerValue, err := flagset.ReadConfigString(cmd, "git-provider")
		if err != nil {
			return err
		}

		if providerValue == "github" {
			if os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN") != "" {
				viper.Set("github.token", os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN"))
			} else {
				log.Fatal("cannot create a cluster without a github auth token. please export your KUBEFIRST_GITHUB_AUTH_TOKEN in your terminal.")
			}
		}

		var globalFlags flagset.GlobalFlags
		var installerFlags flagset.InstallerGenericFlags
		var awsFlags flagset.AwsFlags
		var githubFlags flagset.GithubAddCmdFlags

		if cloudValue == pkg.CloudK3d {

			globalFlags, _, installerFlags, awsFlags, err = flagset.InitFlags(cmd)
			viper.Set("gitops.branch", "main")
			viper.Set("github.owner", viper.GetString("github.user"))
			viper.WriteConfig()

			if installerFlags.BranchGitops = viper.GetString("gitops.branch"); err != nil {
				return err
			}
			if installerFlags.BranchMetaphor = viper.GetString("metaphor.branch"); err != nil {
				return err
			}
			if githubFlags.GithubOwner = viper.GetString("github.owner"); err != nil {
				return err
			}

			if githubFlags.GithubUser = viper.GetString("github.user"); err != nil {
				return err
			}
		} else {
			// github or gitlab
			globalFlags, githubFlags, installerFlags, awsFlags, err = flagset.InitFlags(cmd)
		}
		if err != nil {
			return err
		}

		if globalFlags.SilentMode {
			informUser(
				"Silent mode enabled, most of the UI prints wont be showed. Please check the logs for more details.\n",
				globalFlags.SilentMode,
			)
		}

		if len(awsFlags.AssumeRole) > 0 {
			log.Println("calling assume role")
			err := aws.AssumeRole(awsFlags.AssumeRole)
			if err != nil {
				log.Println(err)
				return err
			}
			log.Printf("assuming new AWS credentials based on role %q", awsFlags.AssumeRole)
		}
		if installerFlags.Cloud == flagset.CloudAws {
			progressPrinter.AddTracker("step-account", pkg.GetAccountInfo, 1)
			progressPrinter.AddTracker("step-dns", pkg.GetDNSInfo, 1)
			progressPrinter.AddTracker("step-live", pkg.TestHostedZoneLiveness, 1)
			progressPrinter.AddTracker("step-buckets", pkg.CreateBuckets, 1)
		}

		progressPrinter.AddTracker("step-download", pkg.DownloadDependencies, 3)
		progressPrinter.AddTracker("step-gitops", pkg.CloneAndDetokenizeGitOpsTemplate, 1)
		progressPrinter.AddTracker("step-ssh", pkg.CreateSSHKey, 1)
		progressPrinter.AddTracker("step-telemetry", pkg.SendTelemetry, 1)

		progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), globalFlags.SilentMode)

		log.Println("sending init started metric")

		var telemetryHandler handlers.TelemetryHandler
		if globalFlags.UseTelemetry {

			// Instantiates a SegmentIO client to use send messages to the segment API.
			segmentIOClient := analytics.New(pkg.SegmentIOWriteKey)

			// SegmentIO library works with queue that is based on timing, we explicit close the http client connection
			// to force flush in case there is still some pending message in the SegmentIO library queue.
			defer func(segmentIOClient analytics.Client) {
				err := segmentIOClient.Close()
				if err != nil {
					log.Println(err)
				}
			}(segmentIOClient)

			// validate telemetryDomain data
			telemetryDomain, err := domain.NewTelemetry(
				pkg.MetricInitStarted,
				awsFlags.HostedZoneName,
				configs.K1Version,
			)
			if err != nil {
				log.Println(err)
			}
			telemetryService := services.NewSegmentIoService(segmentIOClient)
			telemetryHandler = handlers.NewTelemetryHandler(telemetryService)

			err = telemetryHandler.SendCountMetric(telemetryDomain)
			if err != nil {
				log.Println(err)
			}
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

		atlantisWebhookSecret := pkg.Random(20)
		viper.Set("github.atlantis.webhook.secret", atlantisWebhookSecret)

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
		if installerFlags.Cloud == flagset.CloudLocal {
			err = downloadManager.DownloadLocalTools(config)
			if err != nil {
				return err
			}
		}

		//Fix incomplete bar, please don't remove it.
		progressPrinter.IncrementTracker("step-download", 1)

		if installerFlags.Cloud == flagset.CloudAws {
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
		}
		//! tracker 5
		log.Println("creating an ssh key pair for your new cloud infrastructure")
		pkg.CreateSshKeyPair()
		log.Println("ssh key pair creation complete")
		progressPrinter.IncrementTracker("step-ssh", 1)

		//! tracker 6
		repo.PrepareKubefirstTemplateRepo(globalFlags.DryRun, config, viper.GetString("gitops.owner"), viper.GetString("gitops.repo"), viper.GetString("gitops.branch"), viper.GetString("template.tag"))
		log.Println("clone and detokenization of gitops-template repository complete")
		progressPrinter.IncrementTracker("step-gitops", 1)

		log.Println("sending init completed metric")

		if globalFlags.UseTelemetry {
			telemetryInitCompleted, err := domain.NewTelemetry(
				pkg.MetricInitCompleted,
				awsFlags.HostedZoneName,
				configs.K1Version,
			)
			if err != nil {
				log.Println(err)
			}
			err = telemetryHandler.SendCountMetric(telemetryInitCompleted)
			if err != nil {
				log.Println(err)
			}
		}

		viper.WriteConfig()

		//! tracker 8
		progressPrinter.IncrementTracker("step-telemetry", 1)
		time.Sleep(time.Millisecond * 100)

		informUser("init is done!\n", globalFlags.SilentMode)

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

	//validations happens on /internal/flagset
}
