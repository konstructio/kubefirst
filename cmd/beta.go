/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"

	"github.com/konstructio/kubefirst/cmd/akamai"
	"github.com/konstructio/kubefirst/cmd/google"
	"github.com/konstructio/kubefirst/cmd/k3s"
	"github.com/konstructio/kubefirst/cmd/vultr"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/spf13/cobra"
)

// betaCmd represents the beta command tree
var betaCmd = &cobra.Command{
	Use:   "beta",
	Short: "access kubefirst beta features",
	Long:  `access kubefirst beta features`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("To learn more about kubefirst, run:")
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
		k3s.NewCommand(),
		google.NewCommand(),
		vultr.NewCommand(),
	)
}
