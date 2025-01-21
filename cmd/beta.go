/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// betaCmd represents the beta command tree
func NewBetaCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "beta",
		Short: "access Kubefirst beta features",
		Long:  `access Kubefirst beta features`,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("To learn more about Kubefirst, run:")
			fmt.Println("  kubefirst help")
		},
	}

	return cmd
}
