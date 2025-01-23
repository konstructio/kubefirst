/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/konstructio/kubefirst-api/pkg/configs"
	"github.com/spf13/cobra"
)

func NewVersionCommand() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "print the version number for kubefirst-cli",
		Long:  `All software has versions. This is kubefirst's`,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("### kubefirst-cli golang utility version: %q", configs.K1Version))
		},
	}

	return versionCmd
}
