/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"github.com/spf13/cobra"
)

func NewLocalCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "local",
		Short: "kubefirst local installation with k3d",
		Long:  "kubefirst local installation with k3d",
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
