package cmd

import (
	"fmt"
	"os"

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

		log.Info().Msg("resetting `$HOME/.kubefirst` config")
		viper.Set("argocd", "")
		viper.Set("github", "")
		viper.Set("gitlab", "")
		viper.Set("components", "")
		viper.Set("kbot", "")
		viper.Set("kubefirst-checks", "")
		viper.Set("kubefirst", "")
		viper.WriteConfig()

		if _, err := os.Stat(k1Dir + "/kubeconfig"); !os.IsNotExist(err) {
			err = os.Remove(k1Dir + "/kubeconfig")
			if err != nil {
				return fmt.Errorf("unable to delete %q folder, error: %s", k1Dir+"/kubeconfig", err)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
}
