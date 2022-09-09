package flagset

import (
	"log"

	"github.com/spf13/cobra"
)

// CreateFlags - Create flags
type CreateFlags struct {
	EnableConsole bool
}

// DefineCreateFlags - Define create flags of non-default behaviors or experimental features
func DefineCreateFlags(currentCommand *cobra.Command) {
	currentCommand.Flags().Bool("enable-console", false, "If hand-off screen will be presented on a browser UI")
}

// ProcessCreateFlags - process create flags for experimental features
func ProcessCreateFlags(cmd *cobra.Command) (CreateFlags, error) {
	flags := CreateFlags{}
	enableConsole, err := ReadConfigBool(cmd, "enable-console")
	if err != nil {
		log.Printf("Error Processing - enable-console flag, error: %v", err)
		return flags, err
	}
	flags.EnableConsole = enableConsole
	return flags, nil

}
