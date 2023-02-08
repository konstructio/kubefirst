package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kubefirst/kubefirst/internal/wrappers"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/internal/ssh"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/downloadManager"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/repo"
	"github.com/kubefirst/kubefirst/internal/services"
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

		// github or gitlab
		globalFlags, _, installerFlags, awsFlags, err := flagset.InitFlags(cmd)

		if err != nil {
			return err
		}

		providerValue, err := flagset.ReadConfigString(cmd, "git-provider")
		if err != nil {
			return err
		}

		gitHubAccessToken := config.GithubToken
		httpClient := http.DefaultClient
		gitHubService := services.NewGitHubService(httpClient)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)
		if providerValue == pkg.GitHubProviderName && gitHubAccessToken == "" {

			gitHubAccessToken, err = gitHubHandler.AuthenticateUser()
			if err != nil {
				return err
			}

			if gitHubAccessToken == "" {
				return errors.New("cannot create a cluster without a github auth token. please export your " +
					"KUBEFIRST_GITHUB_AUTH_TOKEN in your terminal",
				)
			}

			// todo: set common way to load env. values (viper->struct->load-env)
			if err := os.Setenv("KUBEFIRST_GITHUB_AUTH_TOKEN", gitHubAccessToken); err != nil {
				return err
			}
			log.Info().Msg("KUBEFIRST_GITHUB_AUTH_TOKEN set via OAuth")
		}

		// get GitHub data to set user and owner based on the provided token
		if providerValue == pkg.GitHubProviderName {
			githubUser, err := gitHubHandler.GetGitHubUser(gitHubAccessToken)
			if err != nil {
				return err
			}

			viper.Set("github.user", githubUser)
			err = viper.WriteConfig()
			if err != nil {
				return err
			}
		}

		if globalFlags.SilentMode {
			informUser(
				"Silent mode enabled, most of the UI prints wont be showed. Please check the logs for more details.\n",
				globalFlags.SilentMode,
			)
		}

		if viper.GetString("cloud") != pkg.CloudAws {
			log.Info().Msg("Not cloud mode attempt to create using cloud cli")
			if err != nil {
				return fmt.Errorf("not support mode of install via this command, only cloud install supported")
			}
		}

		if len(awsFlags.AssumeRole) > 0 {
			log.Info().Msg("calling assume role")
			err := aws.AssumeRole(awsFlags.AssumeRole)
			if err != nil {
				log.Warn().Msgf("%s", err)
				return err
			}
			log.Info().Msgf("assuming new AWS credentials based on role %q", awsFlags.AssumeRole)
		}
		if installerFlags.Cloud == pkg.CloudAws {
			progressPrinter.AddTracker("step-account", pkg.GetAccountInfo, 1)
			progressPrinter.AddTracker("step-dns", pkg.GetDNSInfo, 1)
			progressPrinter.AddTracker("step-live", pkg.TestHostedZoneLiveness, 1)
			progressPrinter.AddTracker("step-buckets", pkg.CreateBuckets, 1)
		}

		progressPrinter.AddTracker("step-download", pkg.DownloadDependencies, 3)
		progressPrinter.AddTracker("step-gitops", pkg.CloneAndDetokenizeGitOpsTemplate, 1)
		progressPrinter.AddTracker("step-ssh", pkg.CreateSSHKey, 1)

		progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), globalFlags.SilentMode)

		log.Info().Msg("sending init started metric")

		viper.Set("use-telemetry", globalFlags.UseTelemetry)

		gitProvider := viper.GetString("git-provider")
		cloud := viper.GetString("cloud")

		if !globalFlags.UseTelemetry {
			informUser("Telemetry Disabled", globalFlags.SilentMode)
		} else {
			pkg.InformUser("Sending installation telemetry", globalFlags.SilentMode)
			if err := wrappers.SendSegmentIoTelemetry(awsFlags.HostedZoneName, pkg.MetricInitStarted, cloud, gitProvider); err != nil {
				log.Warn().Msgf("%s", err)
			}
		}

		// todo need to check flags and create config

		// hosted zone name:
		// name of the hosted zone to be used for the kubefirst install
		// if suffixed with a dot (eg. kubefirst.com.), the dot will be stripped
		if strings.HasSuffix(awsFlags.HostedZoneName, ".") {
			awsFlags.HostedZoneName = awsFlags.HostedZoneName[:len(awsFlags.HostedZoneName)-1]
		}
		log.Info().Msgf("hostedZoneName: %s", awsFlags.HostedZoneName)

		viper.Set("argocd.local.service", "http://localhost:8080")
		viper.Set("gitlab.local.service", "http://localhost:8888")
		viper.Set("vault.local.service", "http://localhost:8200")
		// used for letsencrypt notifications and the gitlab root account

		log.Info().Msgf("s3-suffix: %s", installerFlags.ClusterName)

		atlantisWebhookSecret := pkg.Random(20)
		viper.Set("github.atlantis.webhook.secret", atlantisWebhookSecret)
		currentOwner, err := flagset.ReadConfigString(cmd, "github-owner")
		if err != nil {
			return err
		}
		viper.Set("github.owner", currentOwner)
		viper.WriteConfig()

		//! tracker 0
		log.Info().Msg("installing kubefirst dependencies")
		progressPrinter.IncrementTracker("step-download", 1)
		err = downloadManager.DownloadTools(config)
		if err != nil {
			return err
		}
		log.Info().Msg("dependency installation complete")
		progressPrinter.IncrementTracker("step-download", 1)
		if installerFlags.Cloud == pkg.CloudK3d {
			err = downloadManager.DownloadLocalTools(config)
			if err != nil {
				return err
			}
		}

		//Fix incomplete bar, please don't remove it.
		progressPrinter.IncrementTracker("step-download", 1)

		if installerFlags.Cloud == pkg.CloudAws {
			//! tracker 1
			log.Info().Msg("getting aws account information")
			aws.GetAccountInfo()
			log.Info().Msgf("aws account id: %s\naws user arn: %s", viper.GetString("aws.accountid"), viper.GetString("aws.userarn"))
			progressPrinter.IncrementTracker("step-account", 1)

			//! tracker 2
			// hosted zone id
			// So we don't have to keep looking it up from the domain name to use it
			hostedZoneId := aws.GetDNSInfo(awsFlags.HostedZoneName)
			// viper values set in above function
			log.Info().Msgf("hostedZoneId: %s", hostedZoneId)
			progressPrinter.IncrementTracker("step-dns", 1)

			//! tracker 3
			// todo: this doesn't default to testing the dns check
			skipHostedZoneCheck := viper.GetBool("init.hostedzonecheck.enabled")
			if !skipHostedZoneCheck {
				hostedZoneLiveness := aws.TestHostedZoneLiveness(globalFlags.DryRun, awsFlags.HostedZoneName, hostedZoneId)
				if !hostedZoneLiveness {
					msg := "failed to check the liveness of the HostedZone. A valid public HostedZone on the same AWS " +
						"account as the one where Kubefirst will be installed is required for this operation to " +
						"complete.\nTroubleshoot Steps:\n\n - Make sure you are using the correct AWS account and " +
						"region.\n - Verify that you have the necessary permissions to access the hosted zone.\n - Check " +
						"that the hosted zone is correctly configured and is a public hosted zone\n - Check if the " +
						"hosted zone exists and has the correct name and domain.\n - If you don't have a HostedZone," +
						"please follow these instructions to create one: " +
						"https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/hosted-zones-working-with.html \n\n" +
						"if you are still facing issues please reach out to support team for further assistance"
					log.Error().Msg(msg)
					return errors.New(msg)
				}
			} else {
				log.Info().Msg("skipping hosted zone check")
			}
			progressPrinter.IncrementTracker("step-live", 1)

			//! tracker 4
			//* should we consider going down to a single bucket
			//* for state and artifacts on open source?
			//* hitting a bucket limit on an install might deter someone
			log.Info().Msg("creating buckets for state and artifacts")
			aws.BucketRand(globalFlags.DryRun)
			progressPrinter.IncrementTracker("step-buckets", 1)
			log.Info().Msg("BucketRand() complete")
		}
		//! tracker 5
		log.Info().Msg("creating an ssh key pair for your new cloud infrastructure")
		ssh.CreateSshKeyPair()
		log.Info().Msg("ssh key pair creation complete")
		progressPrinter.IncrementTracker("step-ssh", 1)

		//! tracker 6
		repo.PrepareKubefirstTemplateRepo(globalFlags.DryRun, config, viper.GetString("gitops.owner"), viper.GetString("gitops.repo"), viper.GetString("gitops.branch"), viper.GetString("template.tag"))
		log.Info().Msg("clone and detokenization of gitops-template repository complete")
		progressPrinter.IncrementTracker("step-gitops", 1)

		log.Info().Msg("sending init completed metric")

		if globalFlags.UseTelemetry {
			if err := wrappers.SendSegmentIoTelemetry(awsFlags.HostedZoneName, pkg.MetricInitCompleted, "aws", "github"); err != nil {
				log.Warn().Msgf("%s", err)
			}
		}

		viper.WriteConfig()

		// workaround to wait for segmentIo process the message
		time.Sleep(time.Millisecond * 1000)

		informUser("init is done!\n", globalFlags.SilentMode)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	currentCommand := initCmd
	//log.Info().Msg("kubefirst started")
	//log.SetPrefix("LOG: ")
	//log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)

	// Do we need this?
	initCmd.Flags().Bool("clean", false, "delete any local kubefirst content ~/.kubefirst, ~/.k1")

	//Group Flags
	flagset.DefineGlobalFlags(currentCommand)
	flagset.DefineGithubCmdFlags(currentCommand)
	flagset.DefineAWSFlags(currentCommand)
	flagset.DefineInstallerGenericFlags(currentCommand)

	//validations happens on /internal/flagset
}
