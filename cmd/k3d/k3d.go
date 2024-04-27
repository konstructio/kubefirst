/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewK3DCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k3d",
		Short: "kubefirst k3d installation",
		Long:  "kubefirst k3d",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("To learn more about k3d in kubefirst, run:\tkubefirst k3d --help")

			return cmd.Help()
		},
	}

	cmd.AddCommand(
		NewK3dCreateCommand(),
		NewK3dDestroyCommand(),
		NewMkCertCommand(),
		NewRootCredentialCommand(),
		NewVaultUnsealCommand(),
	)

	return cmd
}
