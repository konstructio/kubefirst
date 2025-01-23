/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewLogsCommand() *cobra.Command {
	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "kubefirst real time logs",
		Long:  `kubefirst real time logs`,
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Fprintln(os.Stderr, "### Now tailing kubefirst logs. Press Ctrl+C to stop. ###")
			logPath := viper.GetString("k1-paths.log-file")

			file, err := os.Open(logPath)
			if err != nil {
				return fmt.Errorf("failed to open log file: %w", err)
			}
			defer file.Close()

			for {
				data := make([]byte, 1024)
				n, err := file.Read(data)
				if err == nil {
					fmt.Fprint(os.Stdout, string(data[:n]))
				} else if err != io.EOF {
					return fmt.Errorf("error reading file: %w", err)
				}

				time.Sleep(100 * time.Millisecond)
			}
		},
	}
	return logsCmd
}
