package cmd

import (
	"time"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/internal/k8s"

	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/reports"

	"github.com/kubefirst/kubefirst/pkg"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var postInstallCmd = &cobra.Command{
	Use:   "post-install",
	Short: "starts post install process",
	Long:  "Starts post install process to open the Console UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		// todo: temporary
		//flagset.DefineGlobalFlags(cmd)
		if viper.GetString("cloud") == pkg.CloudK3d {
			cmd.Flags().Bool("enable-console", true, "If hand-off screen will be presented on a browser UI")
		}
		//globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		//if err != nil {
		//	return err
		//}
		globalFlags := flagset.GlobalFlags{DryRun: false, SilentMode: false, UseTelemetry: true}

		createFlags, err := flagset.ProcessCreateFlags(cmd)
		if err != nil {
			return err
		}

		if createFlags.EnableConsole {
			err := k8s.OpenPortForwardForCloudConConsole()
			if err != nil {
				log.Warn().Msgf("%s", err)
			}

			err = pkg.IsConsoleUIAvailable(pkg.KubefirstConsoleLocalURLCloud)
			if err != nil {
				log.Warn().Msgf("%s", err)
			}

			err = pkg.OpenBrowser(pkg.KubefirstConsoleLocalURLCloud)
			if err != nil {
				log.Warn().Msgf("%s", err)
			}

		} else {
			log.Info().Msg("Skipping the presentation of console and api for the handoff screen")
		}

		reports.HandoffScreen(globalFlags.DryRun, globalFlags.SilentMode)

		time.Sleep(time.Millisecond * 2000)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(postInstallCmd)

	// todo: temporary
	//flagset.DefineGlobalFlags(postInstallCmd)
	//postInstallCmd.Flags().Bool("enable-console", true, "If hand-off screen will be presented on a browser UI")
	//flagset.DefineCreateFlags(currentCommand)
}
