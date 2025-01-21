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
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewResetCommand() *cobra.Command {

	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "removes local kubefirst content to provision a new platform",
		Long:  "removes local kubefirst content to provision a new platform",
		RunE: func(_ *cobra.Command, _ []string) error {
			gitProvider := viper.GetString("kubefirst.git-provider")
			cloudProvider := viper.GetString("kubefirst.cloud-provider")

			checksMap := viper.Get("kubefirst-checks")
			switch v := checksMap.(type) {
			case nil:
				//TODO: Handle for non-bubbletea
				// Handle the nil case explicitly
				// message := `# Successfully reset`
				// progress.Success(message)
				return nil
			case string:
				if v == "" {
					log.Info().Msg("checks map is empty, continuing")
				} else {
					return fmt.Errorf("unable to determine contents of kubefirst-checks: unexpected type %T", v)
				}
			case map[string]interface{}:
				checks, err := parseConfigEntryKubefirstChecks(v)
				if err != nil {
					log.Error().Msgf("error occurred during check parsing: %s - resetting directory without checks", err)
				}
				// If destroy hasn't been run yet, reset should fail to avoid orphaned resources
				switch {
				case checks[fmt.Sprintf("terraform-apply-%s", gitProvider)]:
					return fmt.Errorf(
						"it looks like there's an active %s resource deployment - please run `%s destroy` before continuing",
						gitProvider, cloudProvider,
					)
				case checks[fmt.Sprintf("terraform-apply-%s", cloudProvider)]:
					return fmt.Errorf(
						"it looks like there's an active %s installation - please run `%s destroy` before continuing",
						cloudProvider, cloudProvider,
					)
				}
			default:
				return fmt.Errorf("unable to determine contents of kubefirst-checks: unexpected type %T", v)
			}

			homePath, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("unable to get user home directory: %w", err)
			}

			if err := runReset(homePath); err != nil {
				return fmt.Errorf("error during reset operation: %w", err)
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

	//TODO: Handle for non-bubbletea
	// utils.DisplayLogHints()

	// progressPrinter.AddTracker("removing-platform-content", "Removing local platform content", 2)
	// progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	log.Info().Msg("removing previous platform content")

	k1Dir := fmt.Sprintf("%s/.k1", homePath)
	kubefirstConfig := fmt.Sprintf("%s/.kubefirst", homePath)

	if err := utils.ResetK1Dir(k1Dir); err != nil {
		return fmt.Errorf("error resetting k1 directory: %w", err)
	}
	// TODO: Handle for non-bubbletea
	// progressPrinter.IncrementTracker("removing-platform-content")
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

	// TODO: Handle for non-bubbletea
	// progressPrinter.IncrementTracker("removing-platform-content")
	time.Sleep(time.Second * 2)
	return nil
}
