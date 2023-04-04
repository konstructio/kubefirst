package cmd

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/kubefirst/kubefirst/internal/helpers"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/pkg"
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
		checks := parseConfigEntryKubefirstChecks(viper.Get("kubefirst-checks"))

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
		default:
			helpers.DisplayLogHints()

			progressPrinter.AddTracker("removing-platform-content", "Removing local platform content", 2)
			progressPrinter.SetupProgress(progressPrinter.TotalOfTrackers(), false)

			log.Info().Msg("removing previous platform content")

			homePath, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			k1Dir := fmt.Sprintf("%s/.k1", homePath)

			err = pkg.ResetK1Dir(k1Dir)
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
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
}

// parseConfigEntryKubefirstChecks gathers the kubefirst-checks section of the Viper
// config file and parses as a map[string]bool
func parseConfigEntryKubefirstChecks(raw interface{}) map[string]bool {
	checks := raw.(map[string]interface{})
	checksMap := make(map[string]bool, 0)
	for key, value := range checks {
		strKey := fmt.Sprintf("%v", key)
		boolValue := fmt.Sprintf("%v", value)

		boolValueP, _ := strconv.ParseBool(boolValue)
		checksMap[strKey] = boolValueP
	}

	return checksMap
}
