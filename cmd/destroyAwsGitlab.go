/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// destroyAwsGitlabCmd represents the destroyAwsGitlab command
var destroyAwsGitlabCmd = &cobra.Command{
	Use:   "destroy-aws-gitlab",
	Short: "A brief description of your command",
	Long:  `TDB`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Debug().Msg("destroy-aws-gitlab called")

		config := configs.ReadConfig()
		destroyFlags, err := flagset.ProcessDestroyFlags(cmd)
		if err != nil {
			log.Warn().Msgf("%s", err)
			return err
		}

		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			log.Warn().Msgf("%s", err)
			return err
		}

		if globalFlags.SilentMode {
			informUser(
				"Silent mode enabled, most of the UI prints wont be showed. Please check the logs for more details.\n",
				globalFlags.SilentMode,
			)
		}
		//Don't log this, this leaks credentials
		//log.Println(destroyFlags, config)

		progressPrinter.SetupProgress(2, globalFlags.SilentMode)

		if globalFlags.DryRun {
			destroyFlags.SkipGitlabTerraform = true
			destroyFlags.SkipDeleteRegistryApplication = true
			destroyFlags.SkipBaseTerraform = true
		}
		progressPrinter.AddTracker("step-prepare", "Open Ports", 3)
		progressPrinter.AddTracker("step-destroy", "Destroy Cloud", 4)
		progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), globalFlags.SilentMode)

		informUser("Open argocd port-forward", globalFlags.SilentMode)
		progressPrinter.IncrementTracker("step-prepare", 1)

		log.Info().Msg("destroying gitlab terraform")

		progressPrinter.IncrementTracker("step-destroy", 1)
		informUser("Destroying Gitlab", globalFlags.SilentMode)
		if !destroyFlags.SkipGitlabTerraform {
			gitlab.DestroyGitlabTerraform(destroyFlags.SkipGitlabTerraform)
		}
		progressPrinter.IncrementTracker("step-prepare", 1)
		progressPrinter.IncrementTracker("step-destroy", 1)

		log.Info().Msg("gitlab terraform destruction complete")

		//This should wrapped into a function, maybe to move to: k8s.DeleteRegistryApplication
		if !destroyFlags.SkipDeleteRegistryApplication {
			kPortForwardArgocd, _ := k8s.PortForward(globalFlags.DryRun, "svc/argocd-server", config.KubeConfigPath, config.KubectlClientPath, "argocd", "8080:80")
			defer func() {
				if kPortForwardArgocd != nil {
					log.Info().Msg("Closed argocd port forward")
					_ = kPortForwardArgocd.Process.Signal(syscall.SIGTERM)
				}
			}()
			informUser("Open argocd port-forward", globalFlags.SilentMode)
			log.Info().Msg("deleting registry application in argocd")
			// delete argocd registry
			informUser("Destroying Registry Application", globalFlags.SilentMode)
			k8s.DeleteRegistryApplication(destroyFlags.SkipDeleteRegistryApplication)
		}

		progressPrinter.IncrementTracker("step-prepare", 1)
		progressPrinter.IncrementTracker("step-destroy", 1)
		log.Info().Msg("registry application deleted")

		log.Info().Msg("terraform destroy base")
		informUser("Destroying Cluster", globalFlags.SilentMode)
		terraform.DestroyBaseTerraform(destroyFlags.SkipBaseTerraform)
		progressPrinter.IncrementTracker("step-destroy", 1)

		// destroy hosted zone
		if destroyFlags.HostedZoneDelete {
			hostedZone := viper.GetString("aws.hostedzonename")
			awsHandler := handlers.NewAwsHandler(hostedZone, destroyFlags)
			err := awsHandler.HostedZoneDelete()
			if err != nil {
				// if error, just log it
				log.Warn().Msgf("%s", err)
			}
		}

		informUser("All Destroyed", globalFlags.SilentMode)

		log.Info().Msg("terraform base destruction complete")
		fmt.Println("End of execution destroy")
		time.Sleep(time.Millisecond * 100)
		return nil
	},
}

func init() {
	clusterCmd.AddCommand(destroyAwsGitlabCmd)
	currentCommand := destroyAwsGitlabCmd
	flagset.DefineGlobalFlags(currentCommand)
	flagset.DefineDestroyFlags(currentCommand)
}
