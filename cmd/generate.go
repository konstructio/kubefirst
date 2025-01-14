/*
Copyright (C) 2021-2025, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/spf13/cobra"
)

func GenerateCommand() *cobra.Command {
	generateCommand := &cobra.Command{
		Use:   "generate",
		Short: "code generator helpers",
	}

	// wire up new commands
	generateCommand.AddCommand(generateApp())

	return generateCommand
}

func generateApp() *cobra.Command {
	var name string
	var environments []string

	appScaffoldCmd := &cobra.Command{
		Use:              "app-scaffold",
		Short:            "scaffold the gitops application repo",
		TraverseChildren: true,
		Run: func(_ *cobra.Command, _ []string) {
			progress.Success("hello world")
		},
	}

	appScaffoldCmd.Flags().StringVarP(&name, "name", "n", "", "name of the app")
	appScaffoldCmd.MarkFlagRequired("name")
	appScaffoldCmd.Flags().StringSliceVar(&environments, "environments", []string{"development", "staging", "production"}, "environment names to create")

	return appScaffoldCmd
}
