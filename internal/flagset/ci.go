package flagset

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// CIFlags - Global flags
type CIFlags struct {
	BranchCI string
}

// DefineCIFlags - Define global flags
func DefineCIFlags(currentCommand *cobra.Command) {
	currentCommand.Flags().String("ci-branch", "", "version/branch used on git clone for ci setup instruction")
}

// ProcessCIFlags - process global flags shared between commands like silent, dry-run and use-telemetry
func ProcessCIFlags(cmd *cobra.Command) (CIFlags, error) {
	flags := CIFlags{}

	branchCI, err := ReadConfigString(cmd, "ci-branch")
	if err != nil {
		log.Printf("Error Processing - ci-branch flag, error: %v", err)
		return flags, err
	}
	flags.BranchCI = branchCI
	viper.Set("ci.branch", branchCI)

	return flags, nil

}
