/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"

	"github.com/kubefirst/kubefirst/internal/progress"
	"github.com/kubefirst/runtime/configs"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print the version number for kubefirst-cli",
	Long:  `All software has versions. This is kubefirst's`,
	Run: func(cmd *cobra.Command, args []string) {
		versionMsg := `
##
### kubefirst-cli golang utility version:` + fmt.Sprintf("`%s`", configs.K1Version)

		progress.Success(versionMsg)
	},
}
