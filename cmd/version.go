/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"github.com/konstructio/kubefirst-api/pkg/configs"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print the version number for kubefirst-cli",
	Long:  `All software has versions. This is kubefirst's`,
	Run: func(_ *cobra.Command, _ []string) {
		versionMsg := `
##
### kubefirst-cli golang utility version: ` + configs.K1Version

		progress.Success(versionMsg)
	},
}
