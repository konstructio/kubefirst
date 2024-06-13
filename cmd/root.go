/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kubefirst/kubefirst/cmd/aws"
	"github.com/kubefirst/kubefirst/cmd/civo"
	"github.com/kubefirst/kubefirst/cmd/digitalocean"
	"github.com/kubefirst/kubefirst/cmd/k3d"
	"github.com/kubefirst/kubefirst/internal/common"
	"github.com/kubefirst/kubefirst/internal/logging"
	"github.com/kubefirst/kubefirst/internal/progress"
	"github.com/kubefirst/runtime/configs"
	"github.com/kubefirst/runtime/pkg/progressPrinter"
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubefirst",
		Short: "kubefirst management cluster installer base command",
		Long: `kubefirst management cluster installer provisions an
	open source application delivery platform in under an hour. 
	checkout the docs at docs.kubefirst.io.`,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return configs.InitializeViperConfig(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("To learn more about kubefirst, run:\n\tkubefirst help")

			return cmd.Help()
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			// gracefully quit bubbletea if in progress
			if progress.Progress != nil {
				progress.Progress.Quit()
			}

			return nil
		},
	}

	cmd.AddCommand(
		betaCmd,
		infoCmd,
		resetCmd,
		versionCmd,
		aws.NewCommand(),
		civo.NewCommand(),
		digitalocean.NewCommand(),
		k3d.NewK3DCommand(),
		k3d.NewLocalCommand(),
		LaunchCommand(),
		LetsEncryptCommand(),
		TerraformCommand(),
	)

	cobra.OnInitialize(
		common.CheckForVersionUpdate,
		logging.Init,
	)

	return cmd
}

func Execute() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	progressPrinter.GetInstance()
	progress.InitializeProgressTerminal(ctx)

	errCh := make(chan error, 2)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := NewRootCommand().ExecuteContext(ctx); err != nil {
			errCh <- fmt.Errorf("error executing command: %s", err.Error())
		}

		errCh <- nil
	}()

	go func() {
		if _, err := progress.Progress.Run(); err != nil {
			errCh <- fmt.Errorf("error initializing TUI: %s", err.Error())
		}

		errCh <- nil
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case sig := <-signals:
		fmt.Println("Received signal: ", sig)
		cancel()
	case <-ctx.Done():
		fmt.Println("Finished.")
	}

	fmt.Println("Exiting.")

	return nil
}
