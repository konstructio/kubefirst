/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"

	"github.com/kubefirst/kubefirst-api/pkg/vault"
	"github.com/kubefirst/kubefirst/internal/progress"
	"github.com/spf13/cobra"
)

var (
	vaultURLFlag   string
	vaultTokenFlag string
	outputFileFlag string
)

func TerraformCommand() *cobra.Command {
	terraformCommand := &cobra.Command{
		Use:   "terraform",
		Short: "interact with terraform",
		Long:  "interact with terraform",
	}

	// wire up new commands
	terraformCommand.AddCommand(terraformSetEnv())

	return terraformCommand
}

// terraformSetEnv retrieves Vault secrets and formats them for export in the local
// shell for use with terraform commands
func terraformSetEnv() *cobra.Command {
	terraformSetCmd := &cobra.Command{
		Use:              "set-env",
		Short:            "retrieve data from a target vault secret and format it for use in the local shell via environment variables",
		TraverseChildren: true,
		Run: func(cmd *cobra.Command, args []string) {
			v := vault.VaultConfiguration{
				Config: vault.NewVault(),
			}

			err := v.IterSecrets(vaultURLFlag, vaultTokenFlag, outputFileFlag)
			if err != nil {
				progress.Error(fmt.Sprintf("error during vault read: %s", err))
			}

			message := `
##
### Generated env file at` + fmt.Sprintf("`%s`", outputFileFlag) + `

:bulb: Run` + fmt.Sprintf("`source %s`", outputFileFlag) + ` to set environment variables

`
			progress.Success(message)
		},
	}

	terraformSetCmd.Flags().StringVar(&vaultURLFlag, "vault-url", "", "the URL of the vault instance (required)")
	terraformSetCmd.Flags().StringVar(&vaultTokenFlag, "vault-token", "", "the vault token (required)")
	terraformSetCmd.Flags().StringVar(&outputFileFlag, "output-file", ".env", "the file that will be created in the local directory containing secrets (.env by default)")

	return terraformSetCmd
}
