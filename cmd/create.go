package cmd

import (
	"errors"
	"os/exec"
	"syscall"

	"log"
	"time"

	"github.com/kubefirst/kubefirst/configs"
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

		hostedZoneName := viper.GetString("aws.hostedzonename")

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

		if !viper.GetBool("kubefirst.done") {
			if viper.GetBool("github.enabled") {
				log.Println("Installing Github version of Kubefirst")
				viper.Set("git.mode", "github")
				if viper.GetString("cloud") == flagset.CloudLocal {
					// if not local it is AWS for now
					err := createGithubK3dCmd.RunE(cmd, args)
					if err != nil {
						return err
					}
				} else {
					// if not local it is AWS for now
					err := createGithubCmd.RunE(cmd, args)
					if err != nil {
						return err
					}
				}

			} else {
				log.Println("Installing GitLab version of Kubefirst")
				viper.Set("git.mode", "gitlab")
				if viper.GetString("cloud") == flagset.CloudLocal {
					// We don't support gitlab on local yet
					return errors.New("gitlab is not supported on kubefirst local")

				} else {
					// if not local it is AWS for now
					err := createGitlabCmd.RunE(cmd, args)
					if err != nil {
						return err
					}
				}
			}
			viper.Set("kubefirst.done", true)
			viper.WriteConfig()
		} else {
			log.Println("already executed create command, continuing for readiness checks")
		}

		//! keep eyes here chartmuseum health check
		if viper.GetString("cloud") == flagset.CloudLocal {
			if !viper.GetBool("chartmuseum.host.resolved") {

				//* establish port-forward
				var kPortForwardChartMuseum *exec.Cmd
				kPortForwardChartMuseum, err = k8s.PortForward(globalFlags.DryRun, "chartmuseum", "svc/chartmuseum", "8181:8080")
				defer func() {
					err = kPortForwardChartMuseum.Process.Signal(syscall.SIGTERM)
					if err != nil {
						log.Println("Error closing kPortForwardChartMuseum")
					}
				}()
				pkg.AwaitHostNTimes("http://localhost:8181/health", 5, 5)
				viper.Set("chartmuseum.host.resolved", true)
				viper.WriteConfig()
			} else {
				log.Println("already resolved host for chartmuseum, continuing")
			}

		} else {
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
		}
		//! keep eyes next deploy metaphor
		if viper.GetString("cloud") == flagset.CloudLocal {
			log.Println("Hard break as we are still testing this mode")
			return nil
		}

		informUser("Deploying metaphor applications", globalFlags.SilentMode)
		err = deployMetaphorCmd.RunE(cmd, args)
		if err != nil {
			informUser("Error deploy metaphor applications", globalFlags.SilentMode)
			log.Println("Error running deployMetaphorCmd")
			return err
		}
		err = state.UploadKubefirstToStateStore(globalFlags.DryRun)
		if err != nil {
			log.Println(err)
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
