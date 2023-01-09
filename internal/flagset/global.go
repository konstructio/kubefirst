package flagset

import (
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GlobalFlags - Global flags
type GlobalFlags struct {
	DryRun           bool
	DisableTelemetry bool
	SilentMode       bool
	ConfigFile       string
}

// DefineGlobalFlags - Define global flags
func DefineGlobalFlags(currentCommand *cobra.Command) {
	currentCommand.Flags().Bool("disable-telemetry", false, "installer won't send telemetry data if --disable-telemetry is set")
	currentCommand.Flags().Bool("dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
	currentCommand.Flags().Bool("silent", false, "enable silent mode will make the UI return less content to the screen")
	currentCommand.Flags().StringP("config", "c", "", "File to be imported to bootstrap configs")
	viper.BindPFlag("config.file", currentCommand.Flags().Lookup("config-load"))
}

// ProcessGlobalFlags - process global flags shared between commands like silent, dry-run and disable-telemetry
func ProcessGlobalFlags(cmd *cobra.Command) (GlobalFlags, error) {
	flags := GlobalFlags{}
	config, err := ReadConfigString(cmd, "config")
	if err != nil {
		log.Warn().Msgf("Error Processing - config flag, error: %v", err)
		return flags, err
	}
	flags.ConfigFile = config
	log.Info().Msgf("import config source: %s", flags.ConfigFile)
	if flags.ConfigFile != "" {
		InjectConfigs(flags.ConfigFile)
	}
	dryRun, err := ReadConfigBool(cmd, "dry-run")
	if err != nil {
		log.Warn().Msgf("Error Processing - dry-run flag, error: %v", err)
		return flags, err
	}
	flags.DryRun = dryRun

	disableTelemetry, err := ReadConfigBool(cmd, "disable-telemetry")
	if err != nil {
		log.Warn().Msgf("Error Processing - disable-telemetry flag, error: %v", err)
		return flags, err
	}
	flags.DisableTelemetry = disableTelemetry

	silentMode, err := ReadConfigBool(cmd, "silent")
	if err != nil {
		log.Warn().Msgf("Error Processing - silent flag, error: %v", err)
		return flags, err
	}

	flags.SilentMode = silentMode

	return flags, nil

}
