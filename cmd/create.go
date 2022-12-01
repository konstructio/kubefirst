package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"log"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/domain"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/segmentio/analytics-go"

	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/state"

	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates a Kubefirst management cluster",
	Long: `Based on Kubefirst init command, that creates the Kubefirst configuration file, this command start the
cluster provisioning process spinning up the services, and validates the liveness of the provisioned services.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		defer func() {
			//The goal of this code is to track create time, if it works or not.
			//In the future we can add telemetry signal from these action, to track, success or fail.
			duration := time.Since(start)
			log.Printf("[000] Create duration is %s", duration)

		}()

		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}

		if globalFlags.SilentMode {
			informUser(
				"Silent mode enabled, most of the UI prints wont be showed. Please check the logs for more details.\n",
				globalFlags.SilentMode,
			)
		}

		if viper.GetString("cloud") != flagset.CloudAws {
			log.Println("Not cloud mode attempt to create using cloud cli")
			if err != nil {
				return fmt.Errorf("not support mode of install via this command, only cloud install supported")
			}
		}

		// todo remove this dependency from create.go
		hostedZoneName := viper.GetString("aws.hostedzonename")

		//* telemetry
		if globalFlags.UseTelemetry {
			// Instantiates a SegmentIO client to send messages to the segment API.
			segmentIOClientStart := analytics.New(pkg.SegmentIOWriteKey)

			// SegmentIO library works with queue that is based on timing, we explicit close the http client connection
			// to force flush in case there is still some pending message in the SegmentIO library queue.
			defer func(segmentIOClient analytics.Client) {
				err := segmentIOClient.Close()
				if err != nil {
					log.Println(err)
				}
			}(segmentIOClientStart)

			telemetryDomainStart, err := domain.NewTelemetry(
				pkg.MetricMgmtClusterInstallStarted,
				hostedZoneName,
				configs.K1Version,
			)
			if err != nil {
				log.Println(err)
			}
			telemetryServiceStart := services.NewSegmentIoService(segmentIOClientStart)
			telemetryHandlerStart := handlers.NewTelemetryHandler(telemetryServiceStart)

			err = telemetryHandlerStart.SendCountMetric(telemetryDomainStart)
			if err != nil {
				log.Println(err)
			}
		}

		httpClient := http.DefaultClient
		gitHubService := services.NewGitHubService(httpClient)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)

		providerValue := viper.GetString("gitprovider")

		config := configs.ReadConfig()
		gitHubAccessToken := config.GitHubPersonalAccessToken
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
			log.Println("\nKUBEFIRST_GITHUB_AUTH_TOKEN set via OAuth")
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

		if !viper.GetBool("kubefirst.done") {
			if viper.GetString("gitprovider") == "github" {
				log.Println("Installing Github version of Kubefirst")
				viper.Set("git.mode", "github")
				// if not local it is AWS for now
				err := createGithubCmd.RunE(cmd, args)
				if err != nil {
					return err
				}

			} else {
				log.Println("Installing GitLab version of Kubefirst")
				viper.Set("git.mode", "gitlab")
				// if not local it is AWS for now
				err := createGitlabCmd.RunE(cmd, args)
				if err != nil {
					return err
				}

			}
			viper.Set("kubefirst.done", true)
			viper.WriteConfig()
		} else {
			log.Println("already executed create command, continuing for readiness checks")
		}

		// Relates to issue: https://github.com/kubefirst/kubefirst/issues/386
		// Metaphor needs chart museum for CI works
		informUser("Waiting chartmuseum", globalFlags.SilentMode)
		for i := 1; i < 10; i++ {
			chartMuseum := gitlab.AwaitHostNTimes("chartmuseum", globalFlags.DryRun, 20)
			if chartMuseum {
				informUser("Chartmuseum DNS is ready", globalFlags.SilentMode)
				break
			}
		}
		informUser("Removing self-signed Argo certificate", globalFlags.SilentMode)
		clientset, err := k8s.GetClientSet(globalFlags.DryRun)
		if err != nil {
			log.Printf("Failed to get clientset for k8s : %s", err)
			return err
		}
		argocdPodClient := clientset.CoreV1().Pods("argocd")
		err = k8s.RemoveSelfSignedCertArgoCD(argocdPodClient)
		if err != nil {
			log.Printf("Error removing self-signed certificate from ArgoCD: %s", err)
		}

		informUser("Checking if cluster is ready for use by metaphor apps", globalFlags.SilentMode)
		for i := 1; i < 10; i++ {
			err = k1ReadyCmd.RunE(cmd, args)
			if err != nil {
				log.Println(err)
			} else {
				break
			}
		}

		informUser("Deploying metaphor applications", globalFlags.SilentMode)
		err = deployMetaphorCmd.RunE(cmd, args)
		if err != nil {
			informUser("Error deploy metaphor applications", globalFlags.SilentMode)
			log.Println("Error running deployMetaphorCmd")
			return err
		}

		if viper.GetString("cloud") == flagset.CloudAws {
			//POST-install aws cloud census
			elbName, sg := aws.GetELBByClusterName(viper.GetString("cluster-name"))
			viper.Set("aws.vpcid", aws.GetVPCIdByClusterName(viper.GetString("cluster-name")))
			viper.Set("aws.elb.name", elbName)
			viper.Set("aws.elb.sg", sg)
			viper.WriteConfig()

			err = state.UploadKubefirstToStateStore(globalFlags.DryRun)
			if err != nil {
				log.Println(err)
			}
		}

		log.Println("sending mgmt cluster install completed metric")

		if globalFlags.UseTelemetry {
			// Instantiates a SegmentIO client to send messages to the segment API.
			segmentIOClientCompleted := analytics.New(pkg.SegmentIOWriteKey)

			// SegmentIO library works with queue that is based on timing, we explicit close the http client connection
			// to force flush in case there is still some pending message in the SegmentIO library queue.
			defer func(segmentIOClientCompleted analytics.Client) {
				err := segmentIOClientCompleted.Close()
				if err != nil {
					log.Println(err)
				}
			}(segmentIOClientCompleted)

			telemetryDomainCompleted, err := domain.NewTelemetry(
				pkg.MetricMgmtClusterInstallCompleted,
				hostedZoneName,
				configs.K1Version,
			)
			if err != nil {
				log.Println(err)
			}
			telemetryServiceCompleted := services.NewSegmentIoService(segmentIOClientCompleted)
			telemetryHandlerCompleted := handlers.NewTelemetryHandler(telemetryServiceCompleted)

			err = telemetryHandlerCompleted.SendCountMetric(telemetryDomainCompleted)
			if err != nil {
				log.Println(err)
			}
		}

		log.Println("Kubefirst installation finished successfully")
		informUser("Kubefirst installation finished successfully", globalFlags.SilentMode)

		// todo: temporary code to enable console for localhost
		err = postInstallCmd.RunE(cmd, args)
		if err != nil {
			informUser("Error starting apps from post-install", globalFlags.SilentMode)
			log.Println("Error running postInstallCmd")
			return err
		}

		return nil
	},
}

func init() {
	clusterCmd.AddCommand(createCmd)
	currentCommand := createCmd
	// todo: make this an optional switch and check for it or viper
	createCmd.Flags().Bool("destroy", false, "destroy resources")
	createCmd.Flags().Bool("skip-gitlab", false, "Skip GitLab lab install and vault setup")
	createCmd.Flags().Bool("skip-vault", false, "Skip post-gitClient lab install and vault setup")
	flagset.DefineGlobalFlags(currentCommand)
	flagset.DefineCreateFlags(currentCommand)

}
