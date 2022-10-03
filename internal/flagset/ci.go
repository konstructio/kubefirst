package flagset

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// CIFlags - Global flags
type CIFlags struct {
	BranchCI      string
	DestroyBucket bool
}

// DefineCIFlags - Define global flags
func DefineCIFlags(currentCommand *cobra.Command) {
	currentCommand.Flags().String("ci-branch", "", "version/branch used on git clone for ci setup instruction")
	currentCommand.Flags().Bool("destroy-bucket", false, "destroy bucket that stores tfstate of CI infra as code")
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

	destroyBucket, err := ReadConfigBool(cmd, "destroy-bucket")
	if err != nil {
		log.Printf("Error Processing - destroy-bucket flag, error: %v", err)
		return flags, err
	}
	flags.DestroyBucket = destroyBucket
	viper.Set("destroy.bucket", destroyBucket)

	return flags, nil

}
