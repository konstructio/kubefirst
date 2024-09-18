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
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/spf13/cobra"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "provides general Kubefirst setup data",
	Long:  `Provides machine data, files and folders paths`,
	RunE: func(_ *cobra.Command, _ []string) error {
		config, err := configs.ReadConfig()
		if err != nil {
			return err
		}

		var buf bytes.Buffer

		tw := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)

		fmt.Fprintln(&buf, "##")
		fmt.Fprintln(&buf, "# Info summary")
		fmt.Fprintln(&buf, "")

		fmt.Fprintf(tw, "Name\tValue\n")
		fmt.Fprintf(tw, "---\t---\n")
		fmt.Fprintf(tw, "Operational System\t%s\n", config.LocalOs)
		fmt.Fprintf(tw, "Architecture\t%s\n", config.LocalArchitecture)
		fmt.Fprintf(tw, "Golang version\t%s\n", runtime.Version())
		fmt.Fprintf(tw, "Kubefirst config file\t%s\n", config.KubefirstConfigFilePath)
		fmt.Fprintf(tw, "Kubefirst config folder\t%s\n", config.K1FolderPath)
		fmt.Fprintf(tw, "Kubefirst Version\t%s\n", configs.K1Version)

		progress.Success(buf.String())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
