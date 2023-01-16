package cmd

import (
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/ssl"
	"github.com/spf13/cobra"
)

// restoreSSLCmd represents the restoreSSL command
var restoreSSLCmd = &cobra.Command{
	Use:   "restoreSSL",
	Short: "Restore SSL certificates from a previous install",
	Long:  `Command used to restore existing saved to recycle certificates on a newer re-installation on an already used domain.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info().Msg("Started restoreSSL")
		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			log.Warn().Msgf("Error restoreSSL global flags: %s", err)
			return err
		}

		//includeMetaphorApps, err := cmd.Flags().GetBool("include-metaphor")
		includeMetaphorApps := true
		if err != nil {
			log.Warn().Msgf("Error restoreSSL: %s", err)
			return err
		}
		log.Info().Msgf("RestoreSSL includeMetaphorApps: %t", includeMetaphorApps)

		err = ssl.RestoreSSL(globalFlags.DryRun, includeMetaphorApps)
		if err != nil {
			fmt.Printf("Bucket not found, missing SSL backup, assuming first installation, error is: %v", err)
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(restoreSSLCmd)
	//restoreSSLCmd.Flags().Bool("include-metaphor", false, "Include Metaphor Apps in process")
	flagset.DefineGlobalFlags(restoreSSLCmd)
}
