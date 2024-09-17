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

	"github.com/konstructio/kubefirst-api/pkg/progressPrinter"
	utils "github.com/konstructio/kubefirst-api/pkg/utils"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// resetCmd represents the reset command
var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "removes local kubefirst content to provision a new platform",
	Long:  "removes local kubefirst content to provision a new platform",
	RunE: func(cmd *cobra.Command, args []string) error {
		gitProvider := viper.GetString("kubefirst.git-provider")
		cloudProvider := viper.GetString("kubefirst.cloud-provider")

		checksMap := viper.Get("kubefirst-checks")
		switch v := checksMap.(type) {
		case nil:
			// Handle the nil case explicitly
			message := `# Succesfully reset`
			progress.Success(message)
			return nil
		case string:
			if v == "" {
				log.Info().Msg("checks map is empty, continuing")
			} else {
				return fmt.Errorf("unable to determine contents of kubefirst-checks")
			}
		case map[string]interface{}:
			checks, err := parseConfigEntryKubefirstChecks(v)
			if err != nil {
				log.Error().Msgf("error: %s - resetting directory without checks", err)
			}
			// If destroy hasn't been run yet, reset should fail to avoid orphaned resources
			switch {
			case checks[fmt.Sprintf("terraform-apply-%s", gitProvider)]:
				return fmt.Errorf(
					"it looks like there's an active %s resource deployment - please run %s destroy before continuing",
					gitProvider,
					cloudProvider,
				)
			case checks[fmt.Sprintf("terraform-apply-%s", cloudProvider)]:
				return fmt.Errorf(
					"it looks like there's an active %s installation - please run `%s destroy` before continuing",
					cloudProvider,
					cloudProvider,
				)
			}
		default:
			return fmt.Errorf("unable to determine contents of kubefirst-checks: unexpected type %T", v)

		}

		runReset()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
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
func runReset() error {
	utils.DisplayLogHints()

	progressPrinter.AddTracker("removing-platform-content", "Removing local platform content", 2)
	progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

	log.Info().Msg("removing previous platform content")

	homePath, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	k1Dir := fmt.Sprintf("%s/.k1", homePath)

	err = utils.ResetK1Dir(k1Dir)
	if err != nil {
		return err
	}
	log.Info().Msg("previous platform content removed")
	progressPrinter.IncrementTracker("removing-platform-content", 1)

	log.Info().Msg("resetting `$HOME/.kubefirst` config")
	viper.Set("argocd", "")
	viper.Set("github", "")
	viper.Set("gitlab", "")
	viper.Set("components", "")
	viper.Set("kbot", "")
	viper.Set("kubefirst-checks", "")
	viper.Set("kubefirst", "")
	viper.Set("secrets", "")
	viper.WriteConfig()

	if _, err := os.Stat(k1Dir + "/kubeconfig"); !os.IsNotExist(err) {
		err = os.Remove(k1Dir + "/kubeconfig")
		if err != nil {
			return fmt.Errorf("unable to delete %q folder, error: %s", k1Dir+"/kubeconfig", err)
		}
	}

	progressPrinter.IncrementTracker("removing-platform-content", 1)
	time.Sleep(time.Second * 2)
	progress.Progress.Quit()

	return nil
}
