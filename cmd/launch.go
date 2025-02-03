/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"bytes"
	"fmt"
	"text/tabwriter"

	"github.com/konstructio/kubefirst/internal/cluster"
	"github.com/konstructio/kubefirst/internal/launch"
	"github.com/konstructio/kubefirst/internal/step"
	"github.com/spf13/cobra"
)

// additionalHelmFlags can optionally pass user-supplied flags to helm
var additionalHelmFlags []string

func LaunchCommand() *cobra.Command {
	launchCommand := &cobra.Command{
		Use:   "launch",
		Short: "create a local k3d cluster and launch the Kubefirst console and API in it",
		Long:  "create a local k3d cluster and launch the Kubefirst console and API in it",
	}

	// wire up new commands
	launchCommand.AddCommand(launchUp(), launchDown(), launchCluster())

	return launchCommand
}

// launchUp creates a new k3d cluster with Kubefirst console and API
func launchUp() *cobra.Command {
	launchUpCmd := &cobra.Command{
		Use:              "up",
		Short:            "launch new console and api instance",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			stepper := step.NewStepFactory(cmd.ErrOrStderr())

			stepper.DisplayLogHints("", 5)

			stepper.NewProgressStep("Launching Console and API")

			if err := launch.Up(cmd.Context(), additionalHelmFlags, false, true); err != nil {
				stepper.FailCurrentStep(err)
				return fmt.Errorf("failed to launch console and api: %w", err)
			}

			stepper.CompleteCurrentStep()

			stepper.InfoStep(step.EmojiTada, "Your kubefirst platform provisioner has been created.")

			return nil
		},
	}

	launchUpCmd.Flags().StringSliceVar(&additionalHelmFlags, "helm-flag", []string{}, "additional helm flag to pass to the launch up command - can be used any number of times")

	return launchUpCmd
}

// launchDown destroys a k3d cluster for Kubefirst console and API
func launchDown() *cobra.Command {
	launchDownCmd := &cobra.Command{
		Use:              "down",
		Short:            "remove console and api instance",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			stepper := step.NewStepFactory(cmd.ErrOrStderr())

			stepper.NewProgressStep("Destroying Console and API")

			if err := launch.Down(false); err != nil {
				wrerr := fmt.Errorf("failed to remove console and api: %w", err)
				stepper.FailCurrentStep(wrerr)
				return wrerr
			}

			stepper.CompleteCurrentStep()

			stepper.InfoStep(step.EmojiTada, "Your kubefirst platform provisioner has been destroyed.")

			return nil
		},
	}

	return launchDownCmd
}

// launchCluster
func launchCluster() *cobra.Command {
	launchClusterCmd := &cobra.Command{
		Use:              "cluster",
		Short:            "interact with clusters created by the Kubefirst console",
		TraverseChildren: true,
	}

	launchClusterCmd.AddCommand(launchListClusters(), launchDeleteCluster())

	return launchClusterCmd
}

// launchListClusters makes a request to the console API to list created clusters
func launchListClusters() *cobra.Command {
	launchListClustersCmd := &cobra.Command{
		Use:              "list",
		Short:            "list clusters created by the Kubefirst console",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			stepper := step.NewStepFactory(cmd.ErrOrStderr())

			clusters, err := cluster.GetClusters()
			if err != nil {
				return fmt.Errorf("error getting clusters: %w", err)
			}

			var buf bytes.Buffer
			tw := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)

			fmt.Fprint(tw, "NAME\tCREATED AT\tSTATUS\tTYPE\tPROVIDER\n")
			for _, cluster := range clusters {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
					cluster.ClusterName,
					cluster.CreationTimestamp,
					cluster.Status,
					cluster.ClusterType,
					cluster.CloudProvider)
			}

			stepper.InfoStepString(buf.String())

			return nil
		},
	}

	return launchListClustersCmd
}

// launchDeleteCluster makes a request to the console API to delete a single cluster
func launchDeleteCluster() *cobra.Command {
	launchDeleteClusterCmd := &cobra.Command{
		Use:              "delete",
		Short:            "delete a cluster created by the Kubefirst console",
		TraverseChildren: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return fmt.Errorf("you must provide a cluster name as the only argument to this command")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			stepper := step.NewStepFactory(cmd.ErrOrStderr())

			stepper.NewProgressStep("Deleting Cluster")

			if len(args) != 1 {
				wrerr := fmt.Errorf("expected 1 argument (cluster name)")
				stepper.FailCurrentStep(wrerr)
				return wrerr
			}

			managedClusterName := args[0]

			err := cluster.DeleteCluster(managedClusterName)
			if err != nil {
				wrerr := fmt.Errorf("failed to delete cluster: %w", err)
				stepper.FailCurrentStep(wrerr)
				return wrerr
			}

			deleteMessage := `
				Submitted request to delete cluster` + fmt.Sprintf("`%s`", managedClusterName) + `
				Follow progress with ` + fmt.Sprintf("`%s`", "kubefirst launch cluster list") + `
			`
			stepper.InfoStepString(deleteMessage)

			return nil
		},
	}

	return launchDeleteClusterCmd
}
