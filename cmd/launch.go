/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"

	"github.com/konstructio/kubefirst/internal/launch"
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
		Run: func(_ *cobra.Command, _ []string) {
			launch.Up(additionalHelmFlags, false, true)
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
		Run: func(_ *cobra.Command, _ []string) {
			launch.Down(false)
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
		Run: func(_ *cobra.Command, _ []string) {
			launch.ListClusters()
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
		Run: func(_ *cobra.Command, args []string) {
			launch.DeleteCluster(args[0])
		},
	}

	return launchDeleteClusterCmd
}
