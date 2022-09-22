package flagset

import (
	"log"

	"github.com/spf13/cobra"
)

// DestroyFlags - Global flags
type DestroyFlags struct {
	SkipGitlabTerraform           bool
	SkipDeleteRegistryApplication bool
	SkipBaseTerraform             bool
}

// DefineDestroyFlags - Define global flags
func DefineDestroyFlags(currentCommand *cobra.Command) {
	currentCommand.Flags().Bool("skip-gitlab-terraform", false, "whether to skip the terraform destroy against gitlab - note: if you already deleted registry it doesnt exist")
	currentCommand.Flags().Bool("skip-delete-register", false, "whether to skip deletion of register application ")
	currentCommand.Flags().Bool("skip-base-terraform", false, "whether to skip the terraform destroy against base install - note: if you already deleted registry it doesnt exist")
}

// ProcessDestroyFlags - process global flags shared between commands like silent, dry-run and use-telemetry
func ProcessDestroyFlags(cmd *cobra.Command) (DestroyFlags, error) {
	flags := DestroyFlags{}

	skipGitlabTerraform, err := ReadConfigBool(cmd, "skip-gitlab-terraform")
	if err != nil {
		log.Printf("Error Processing - skip-gitlab-terraform, error: %v", err)
		return flags, err
	}
	flags.SkipGitlabTerraform = skipGitlabTerraform

	skipDeleteRegistryApplication, err := ReadConfigBool(cmd, "skip-delete-register")
	if err != nil {
		log.Printf("Error Processing - skip-delete-register flag, error: %v", err)
		return flags, err
	}
	flags.SkipDeleteRegistryApplication = skipDeleteRegistryApplication

	skipBaseTerraform, err := ReadConfigBool(cmd, "skip-base-terraform")
	if err != nil {
		log.Printf("Error Processing - skip-base-terraform flag, error: %v", err)
		return flags, err
	}

	flags.SkipBaseTerraform = skipBaseTerraform

	return flags, nil

}
