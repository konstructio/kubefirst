/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"bytes"
	"fmt"
	"runtime"
	"text/tabwriter"

	"github.com/konstructio/kubefirst-api/pkg/configs"
	"github.com/konstructio/kubefirst/internal/step"
	"github.com/spf13/cobra"
)

func InfoCommand() *cobra.Command {
	infoCmd := &cobra.Command{
		Use:   "info",
		Short: "provides general Kubefirst setup data",
		Long:  `Provides machine data, files and folders paths`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			stepper := step.NewStepFactory(cmd.ErrOrStderr())
			config, err := configs.ReadConfig()
			if err != nil {
				wrerr := fmt.Errorf("failed to read config: %w", err)
				stepper.InfoStep(step.EmojiError, wrerr.Error())
				return wrerr
			}

			var buf bytes.Buffer

			tw := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)

			fmt.Fprintln(&buf, "")
			fmt.Fprintln(&buf, "Info summary")
			fmt.Fprintln(&buf, "")

			fmt.Fprintf(tw, "Name\tValue\n")
			fmt.Fprintf(tw, "---\t---\n")
			fmt.Fprintf(tw, "Operational System\t%s\n", config.LocalOs)
			fmt.Fprintf(tw, "Architecture\t%s\n", config.LocalArchitecture)
			fmt.Fprintf(tw, "Golang version\t%s\n", runtime.Version())
			fmt.Fprintf(tw, "Kubefirst config file\t%s\n", config.KubefirstConfigFilePath)
			fmt.Fprintf(tw, "Kubefirst config folder\t%s\n", config.K1FolderPath)
			fmt.Fprintf(tw, "Kubefirst Version\t%s\n", configs.K1Version)
			tw.Flush()

			stepper.InfoStepString(buf.String())
			return nil
		},
	}

	return infoCmd
}
