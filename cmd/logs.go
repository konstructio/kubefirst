/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"

	"github.com/kubefirst/kubefirst/internal/progress"
	"github.com/kubefirst/kubefirst/internal/provisionLogs"
	"github.com/nxadm/tail"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// infoCmd represents the info command
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "kubefirst real time logs",
	Long:  `kubefirst real time logs`,
	RunE: func(cmd *cobra.Command, args []string) error {
		provisionLogs.InitializeProvisionLogsTerminal()

		go func() {
			t, err := tail.TailFile(viper.GetString("k1-paths.log-file"), tail.Config{Follow: true, ReOpen: true})
			if err != nil {
				fmt.Printf("Error tailing log file: %v\n", err)
				progress.Progress.Quit()
			}

			for line := range t.Lines {
				provisionLogs.AddLog(line.Text)
			}
		}()

		provisionLogs.ProvisionLogs.Run()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
