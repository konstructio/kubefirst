package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

		if viper.GetString("cloud") == flagset.CloudK3d {
			// todo need to add go channel to control when ngrok should close
			go pkg.RunNgrok(context.TODO(), pkg.LocalAtlantisURL)
			time.Sleep(5 * time.Second)
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

		informUser("Deploying metaphor applications", globalFlags.SilentMode)
		err = deployMetaphorCmd.RunE(cmd, args)
		if err != nil {
			informUser("Error deploy metaphor applications", globalFlags.SilentMode)
			log.Println("Error running deployMetaphorCmd")
			return err
		}

		if viper.GetString("cloud") == flagset.CloudAws {
			err = state.UploadKubefirstToStateStore(globalFlags.DryRun)
			if err != nil {
				log.Println(err)
			}
		}

		//kPortForwardAtlantis, err := k8s.PortForward(globalFlags.DryRun, "atlantis", "svc/atlantis", "4141:80")
		//defer func() {
		//	err = kPortForwardAtlantis.Process.Signal(syscall.SIGTERM)
		//	if err != nil {
		//		log.Println("error closing kPortForwardAtlantis")
		//	}
		//}()

		// ---
		clientset, err := k8s.GetClientSet(false)
		atlantisSecrets, err := clientset.CoreV1().Secrets("atlantis").Get(context.TODO(), "atlantis-secrets", metav1.GetOptions{})
		if err != nil {
			return err
		}

		// todo: hardcoded
		atlantisSecrets.Data["TF_VAR_vault_addr"] = []byte("http://vault.vault.svc.cluster.local:8200")
		atlantisSecrets.Data["VAULT_ADDR"] = []byte("http://vault.vault.svc.cluster.local:8200")

		_, err = clientset.CoreV1().Secrets("atlantis").Update(context.TODO(), atlantisSecrets, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		err = clientset.CoreV1().Pods("atlantis").Delete(context.TODO(), "atlantis-0", metav1.DeleteOptions{})
		if err != nil {
			log.Fatal(err)
		}
		log.Println("---debug---")
		log.Println("sleeping after kill atlantis pod")
		log.Println("---debug---")

		time.Sleep(10 * time.Second)

		log.Println("---debug---")
		log.Println("new port forward atlantis")
		log.Println("---debug---")
		kPortForwardAtlantis, err := k8s.PortForward(false, "atlantis", "svc/atlantis", "4141:80")
		defer func() {
			err = kPortForwardAtlantis.Process.Signal(syscall.SIGTERM)
			if err != nil {
				log.Println("error closing kPortForwardAtlantis")
			}
		}()

		/*

		 this is atlantis temporary code

		*/
		// 1
		err = pkg.ReplaceS3Backend()
		if err != nil {
			return err
		}
		//
		// 2 git push to new repo
		////ref := plumbing.NewHashReference("refs/heads/update-s3-backend", headRef.Hash())
		//
		githubHost := viper.GetString("github.host")
		githubOwner := viper.GetString("github.owner")
		remoteName := "github"
		localRepo := "gitops"
		branchName := "update-s3-backend"
		branchNameRef := plumbing.ReferenceName("refs/heads/" + branchName)

		gitClient.UpdateLocalTFFilesAndPush(
			githubHost,
			githubOwner,
			localRepo,
			remoteName,
			branchNameRef,
		)

		fmt.Println("sleeping after commit...")
		time.Sleep(3 * time.Second)

		// 3
		g := githubWrapper.New()
		err = g.CreatePR(branchName)
		if err != nil {
			fmt.Println(err)
		}
		log.Println("sleeping after create PR...")
		time.Sleep(5 * time.Second)
		log.Println("sleeping... atlantis plan should be running")
		time.Sleep(5 * time.Second)

		fmt.Println("sleeping before apply...")
		time.Sleep(120 * time.Second)

		// 5
		// call atlantis apply
		err = g.CommentPR(1, "atlantis apply")
		if err != nil {
			fmt.Println(err)
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
