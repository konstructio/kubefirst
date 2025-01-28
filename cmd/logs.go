/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"

	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/konstructio/kubefirst/internal/provisionLogs"
	"github.com/nxadm/tail"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func LogsCommand() *cobra.Command {
	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "kubefirst real time logs",
		Long:  `kubefirst real time logs`,
		RunE: func(_ *cobra.Command, _ []string) error {
			provisionLogs.InitializeProvisionLogsTerminal()

			go func() {
				t, err := tail.TailFile(viper.GetString("k1-paths.log-file"), tail.Config{Follow: true, ReOpen: true})
				if err != nil {
					fmt.Printf("Error tailing log file: %v\n", err)
					progress.Progress.Quit()
					return
				}

				for line := range t.Lines {
					provisionLogs.AddLog(line.Text)
				}
			}()

			if _, err := provisionLogs.ProvisionLogs.Run(); err != nil {
				return fmt.Errorf("failed to run provision logs: %w", err)
			}

			return nil
		},
	}

	return logsCmd
}
