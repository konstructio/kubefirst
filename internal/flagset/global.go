package flagset

import (
	"log"

	"github.com/spf13/cobra"
)

type GlobalFlags struct {
	DryRun       bool
	UseTelemetry bool
}

func DefineGlobalFlags(currentCommand *cobra.Command) {
	currentCommand.Flags().Bool("use-telemetry", true, "installer will not send telemetry about this installation")
	currentCommand.Flags().Bool("dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
}

func ProcessGlobalFlags(cmd *cobra.Command) (GlobalFlags, error) {
	flags := GlobalFlags{}

	dryrun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		log.Println("Error Processing - dry-run flag")
		return flags, err
	}
	flags.DryRun = dryrun

	useTelemetry, err := cmd.Flags().GetBool("use-telemetry")
	if err != nil {
		log.Println("Error Processing - use-telemetry flag")
		return flags, err
	}
	flags.UseTelemetry = useTelemetry

	return flags, nil

}
