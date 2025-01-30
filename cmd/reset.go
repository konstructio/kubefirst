/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cmd

import (
	"fmt"
	"os"
	"strconv"
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
			gitProvider := viper.GetString("kubefirst.git-provider")
			cloudProvider := viper.GetString("kubefirst.cloud-provider")

			checksMap := viper.Get("kubefirst-checks")
			switch v := checksMap.(type) {
			case nil:
				// Handle the nil case explicitly
				message := "Successfully reset kubefirst platform"
				stepper.InfoStep(step.EmojiTada, message)
				return nil
			case string:
				if v == "" {
					log.Info().Msg("checks map is empty, continuing")
				} else {
					wrerr := fmt.Errorf("unexpected string value in kubefirst-checks: %s", v)
					stepper.InfoStep(step.EmojiError, wrerr.Error())
					return wrerr
				}
			case map[string]interface{}:
				checks, err := parseConfigEntryKubefirstChecks(v)
				if err != nil {
					wrerr := fmt.Errorf("error parsing kubefirst-checks: %w", err)
					stepper.InfoStep(step.EmojiError, wrerr.Error())
					log.Error().Msgf("error occurred during check parsing: %s - resetting directory without checks", err)
				}
				// If destroy hasn't been run yet, reset should fail to avoid orphaned resources
				switch {
				case checks[fmt.Sprintf("terraform-apply-%s", gitProvider)]:
					wrerr := fmt.Errorf("active %s resource deployment detected - please run `%s destroy` before continuing", gitProvider, cloudProvider)
					stepper.InfoStep(step.EmojiError, wrerr.Error())
					return wrerr
				case checks[fmt.Sprintf("terraform-apply-%s", cloudProvider)]:
					wrerr := fmt.Errorf("active %s installation detected - please run `%s destroy` before continuing", cloudProvider, cloudProvider)
					stepper.InfoStep(step.EmojiError, wrerr.Error())
					return wrerr
				}
			default:
				wrerr := fmt.Errorf("unable to determine contents of kubefirst-checks: unexpected type %T", v)
				stepper.InfoStep(step.EmojiError, wrerr.Error())
				return wrerr
			}

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
			return nil
		},
	}

	return resetCmd
}

// parseConfigEntryKubefirstChecks gathers the kubefirst-checks section of the Viper
// config file and parses as a map[string]bool
func parseConfigEntryKubefirstChecks(checks map[string]interface{}) (map[string]bool, error) {
	if checks == nil {
		return map[string]bool{}, fmt.Errorf("checks configuration is nil")
	}
	checksMap := make(map[string]bool, 0)
	for key, value := range checks {
		strKey := fmt.Sprintf("%v", key)
		boolValue := fmt.Sprintf("%v", value)

		boolValueP, _ := strconv.ParseBool(boolValue)
		checksMap[strKey] = boolValueP
	}

	return checksMap, nil
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
