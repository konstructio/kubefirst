/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"

	"github.com/konstructio/kubefirst-api/pkg/configs"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/spf13/cobra"
)

var (
	ciFlag bool
)

func init() {
	rootCmd.AddCommand(Create())
}

func Create() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "print the version number for kubefirst-cli",
		Long:  `All software has versions. This is kubefirst's`,
		Run: func(cmd *cobra.Command, args []string) {
			versionMsg := `
##
### kubefirst-cli golang utility version:` + fmt.Sprintf("`%s`", configs.K1Version)

			// canRunBubbleTea := viper.GetBool("k1-canRunBubbleTea")

			// if canRunBubbleTea {
			progress.Success(versionMsg)
			// } else {
			// 	fmt.Print(versionMsg)
			// }
		},
	}

	return versionCmd
}
