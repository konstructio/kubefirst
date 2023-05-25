/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"
	"os"

	k3dint "github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/downloadManager"
	"github.com/kubefirst/runtime/pkg/helpers"
	"github.com/kubefirst/runtime/pkg/k3d"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	clusterName = "kubefirst-console"
)

func LaunchCommand() *cobra.Command {
	launchCommand := &cobra.Command{
		Use:   "launch",
		Short: "create a local k3d cluster and launch the kubefirst console and api in it",
		Long:  "create a local k3d cluster and launch the kubefirst console and api in it",
	}

	// wire up new commands
	launchCommand.AddCommand(launchUp(), launchDown())

	return launchCommand
}

// launchUp creates a new k3d cluster with Kubefirst console and API
func launchUp() *cobra.Command {
	launchUpCmd := &cobra.Command{
		Use:              "up",
		Short:            "launch new console and api instance",
		TraverseChildren: true,
		Run: func(cmd *cobra.Command, args []string) {
			choice := k3dint.MongoDestinationChooser()
			switch choice {
			case "atlas":
			case "in-cluster":
			default:
				log.Fatalf("%s is not a valid option", choice)
			}

			helpers.DisplayLogHints()

			homeDir, err := os.UserHomeDir()
			if err != nil {
				log.Fatalf("something went wrong getting home path: %s", err)
			}
			dir := fmt.Sprintf("%s/.k1/%s", homeDir, clusterName)
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				err := os.MkdirAll(dir, os.ModePerm)
				if err != nil {
					log.Infof("%s directory already exists, continuing", dir)
				}
			}
			toolsDir := fmt.Sprintf("%s/tools", dir)
			if _, err := os.Stat(toolsDir); os.IsNotExist(err) {
				err := os.MkdirAll(toolsDir, os.ModePerm)
				if err != nil {
					log.Infof("%s directory already exists, continuing", toolsDir)
				}
			}

			k3dClient := fmt.Sprintf("%s/k3d", toolsDir)
			k3dDownloadUrl := fmt.Sprintf(
				"https://github.com/k3d-io/k3d/releases/download/%s/k3d-%s-%s",
				k3d.K3dVersion,
				k3d.LocalhostOS,
				k3d.LocalhostARCH,
			)
			err = downloadManager.DownloadFile(k3dClient, k3dDownloadUrl)
			if err != nil {
				log.Fatalf("error while trying to download k3d: %s", err)
			}
			err = os.Chmod(k3dClient, 0755)
			if err != nil {
				log.Fatal(err.Error())
			}

			log.Info("Creating k3d cluster for Kubefirst console and API")
			err = k3d.ClusterCreateConsoleAPI(
				clusterName,
				fmt.Sprintf("%s/.k1/%s/kubeconfig", homeDir, clusterName),
				k3dClient,
				fmt.Sprintf("%s/kubeconfig", dir),
			)
			if err != nil {
				msg := fmt.Sprintf("error creating k3d cluster: %s", err)
				log.Fatal(msg)
			}
			log.Info("k3d cluster for Kubefirst console and API created successfully")

			// Helm install below
		},
	}

	return launchUpCmd
}

// launchDown destroys a k3d cluster for Kubefirst console and API
func launchDown() *cobra.Command {
	launchDownCmd := &cobra.Command{
		Use:              "down",
		Short:            "remove console and api instance",
		TraverseChildren: true,
		Run: func(cmd *cobra.Command, args []string) {
			helpers.DisplayLogHints()

			homeDir, err := os.UserHomeDir()
			if err != nil {
				log.Fatalf("something went wrong getting home path: %s", err)
			}

			log.Info("Deleting k3d cluster for Kubefirst console and API")

			dir := fmt.Sprintf("%s/.k1/%s", homeDir, clusterName)
			toolsDir := fmt.Sprintf("%s/tools", dir)
			k3dClient := fmt.Sprintf("%s/k3d", toolsDir)

			_, _, err = pkg.ExecShellReturnStrings(k3dClient, "cluster", "delete", clusterName)
			if err != nil {
				log.Fatal("error deleting k3d cluster")
			}

			log.Info("k3d cluster for Kubefirst console and API deleted successfully")

			err = os.RemoveAll(dir)
			if err != nil {
				log.Warnf("unable to remove directory at %s", dir)
			}
		},
	}

	return launchDownCmd
}
