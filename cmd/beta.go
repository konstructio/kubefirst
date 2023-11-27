/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"

	"github.com/kubefirst/kubefirst/cmd/digitalocean"
	"github.com/kubefirst/kubefirst/cmd/google"
	"github.com/kubefirst/kubefirst/cmd/vultr"
	"github.com/kubefirst/kubefirst/internal/progress"
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
		digitalocean.NewCommand(),
		google.NewCommand(),
		vultr.NewCommand(),
	)
}
