/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"

	"github.com/konstructio/kubefirst-api/pkg/configs"
	"github.com/konstructio/kubefirst/cmd/aws"
	"github.com/konstructio/kubefirst/cmd/civo"
	"github.com/konstructio/kubefirst/cmd/digitalocean"
	"github.com/konstructio/kubefirst/cmd/k3d"
	"github.com/spf13/cobra"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:   "kubefirst",
		Short: "kubefirst management cluster installer base command",
		Long: `kubefirst management cluster installer provisions an
	open source application delivery platform in under an hour.
	checkout the docs at https://kubefirst.konstruct.io/docs/.`,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// wire viper config for flags for all commands
			return configs.InitializeViperConfig(cmd)
		},
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("To learn more about kubefirst, run:")
			fmt.Println("  kubefirst help")
		},
	}

	rootCmd.SilenceUsage = true
	rootCmd.AddCommand(
		betaCmd,
		aws.NewCommand(),
		civo.NewCommand(),
		digitalocean.NewCommand(),
		k3d.NewCommand(),
		k3d.LocalCommandAlias(),
		LaunchCommand(),
		LetsEncryptCommand(),
		TerraformCommand(),

		// Subcommands
		resetCmd,
		infoCmd,
		logsCmd,
		versionCmd,
	)

	return rootCmd.Execute()
}
