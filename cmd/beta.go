/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"

	"github.com/konstructio/kubefirst/cmd/k3s"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/spf13/cobra"
)

func BetaCommands() *cobra.Command {

	betaCmd := &cobra.Command{
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

	betaCmd.AddCommand(
		k3s.NewCommand(),
	)

	return betaCmd
}
