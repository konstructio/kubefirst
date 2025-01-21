/*
Copyright (C) 2021-2025, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/konstructio/kubefirst/internal/generate"
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
	var outputPath string

	appScaffoldCmd := &cobra.Command{
		Use:              "app-scaffold",
		Short:            "scaffold the gitops application repo",
		TraverseChildren: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := generate.AppScaffold(name, environments, outputPath); err != nil {
				progress.Error(err.Error())
				return fmt.Errorf("error scaffolding app: %w", err)
			}

			progress.Success(fmt.Sprintf("App successfully scaffolded: %s", name))
			return nil
		},
	}

	appScaffoldCmd.Flags().StringVarP(&name, "name", "n", "", "name of the app")
	appScaffoldCmd.MarkFlagRequired("name")
	appScaffoldCmd.Flags().StringSliceVar(&environments, "environments", []string{"development", "staging", "production"}, "environment names to create")
	appScaffoldCmd.Flags().StringVar(&outputPath, "output-path", filepath.Join(".", "registry", "environments"), "location to save generated files")

	return appScaffoldCmd
}
