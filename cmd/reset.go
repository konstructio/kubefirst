/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"
	"os"
	"time"

	utils "github.com/konstructio/kubefirst-api/pkg/utils"
	"github.com/konstructio/kubefirst/internal/step"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func ResetCommand() *cobra.Command {
	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "removes local kubefirst content to provision a new platform",
		Long:  "removes local kubefirst content to provision a new platform",
		RunE: func(cmd *cobra.Command, _ []string) error {
			stepper := step.NewStepFactory(cmd.ErrOrStderr())

			homePath, err := os.UserHomeDir()
			if err != nil {
				wrerr := fmt.Errorf("unable to get user home directory: %w", err)
				stepper.InfoStep(step.EmojiError, wrerr.Error())
				return wrerr
			}

			if err := runReset(homePath); err != nil {
				wrerr := fmt.Errorf("failed to reset kubefirst platform: %w", err)
				stepper.InfoStep(step.EmojiError, wrerr.Error())
				return wrerr
			}

			stepper.InfoStep(step.EmojiTada, "Successfully reset kubefirst platform")

			return nil
		},
	}

	return resetCmd
}

// runReset carries out the reset function
func runReset(homePath string) error {
	log.Info().Msg("removing previous platform content")

	k1Dir := fmt.Sprintf("%s/.k1", homePath)
	kubefirstConfig := fmt.Sprintf("%s/.kubefirst", homePath)

	if err := utils.ResetK1Dir(k1Dir); err != nil {
		return fmt.Errorf("error resetting k1 directory: %w", err)
	}
	log.Info().Msg("previous platform content removed")

	log.Info().Msg("resetting $HOME/.kubefirst config")
	viper.Set("argocd", "")
	viper.Set("github", "")
	viper.Set("gitlab", "")
	viper.Set("components", "")
	viper.Set("kbot", "")
	viper.Set("kubefirst-checks", "")
	viper.Set("kubefirst", "")
	viper.Set("secrets", "")
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("error writing viper config: %w", err)
	}

	if err := os.RemoveAll(k1Dir); err != nil {
		return fmt.Errorf("unable to delete %q folder, error: %w", k1Dir, err)
	}

	if err := os.RemoveAll(kubefirstConfig); err != nil {
		return fmt.Errorf("unable to remove %q, error: %w", kubefirstConfig, err)
	}

	time.Sleep(time.Second * 2)
	return nil
}
