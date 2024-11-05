/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"

	"github.com/konstructio/kubefirst/cmd/akamai"
	"github.com/konstructio/kubefirst/cmd/azure"
	"github.com/konstructio/kubefirst/cmd/google"
	"github.com/konstructio/kubefirst/cmd/k3s"
	"github.com/konstructio/kubefirst/cmd/vultr"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/spf13/cobra"
)

// betaCmd represents the beta command tree
var betaCmd = &cobra.Command{
	Use:   "beta",
	Short: "access Kubefirst beta features",
	Long:  `access Kubefirst beta features`,
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("To learn more about Kubefirst, run:")
		fmt.Println("  kubefirst help")

		if progress.Progress != nil {
			progress.Progress.Quit()
		}
	},
}

func init() {
	cobra.OnInitialize()
	betaCmd.AddCommand(
		akamai.NewCommand(),
		azure.NewCommand(),
		k3s.NewCommand(),
		google.NewCommand(),
		vultr.NewCommand(),
	)
}
