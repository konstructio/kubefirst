package flagset

import (
	"errors"
	"log"

	"github.com/spf13/cobra"
)

// DestroyFlags - Global flags
type DestroyFlags struct {
	SkipGitlabTerraform           bool
	SkipGithubTerraform           bool
	SkipDeleteRegistryApplication bool
	SkipBaseTerraform             bool
	HostedZoneDelete              bool
	HostedZoneKeepBase            bool
}

// DefineDestroyFlags - Define global flags
func DefineDestroyFlags(currentCommand *cobra.Command) {
	currentCommand.Flags().Bool("skip-gitlab-terraform", false, "whether to skip the terraform destroy against gitlab - note: if you already deleted registry it doesnt exist")
	currentCommand.Flags().Bool("skip-github-terraform", false, "whether to skip the terraform destroy against github - note: if you already deleted registry it doesnt exist")
	currentCommand.Flags().Bool("skip-delete-register", false, "whether to skip deletion of register application")
	currentCommand.Flags().Bool("skip-base-terraform", false, "whether to skip the terraform destroy against base install - note: if you already deleted registry it doesnt exist")

	currentCommand.Flags().Bool("hosted-zone-delete", false, "delete full hosted zone, use --keep-base-hosted-zone in combination to keep base DNS records (NS, SOA, liveness)")
	currentCommand.Flags().Bool("hosted-zone-keep-base", false, "keeps base DNS records (NS, SOA and liveness TXT), and delete all other DNS records. Use it in combination with --hosted-zone-delete")
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

	skipGithubTerraform, err := ReadConfigBool(cmd, "skip-github-terraform")
	if err != nil {
		log.Printf("Error Processing - skip-github-terraform, error: %v", err)
		return flags, err
	}
	flags.SkipGithubTerraform = skipGithubTerraform

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

	// group flags for hosted zone
	flags.HostedZoneDelete, err = cmd.Flags().GetBool("hosted-zone-delete")
	if err != nil {
		return DestroyFlags{}, err
	}

	flags.HostedZoneKeepBase, err = cmd.Flags().GetBool("hosted-zone-keep-base")
	if err != nil {
		return DestroyFlags{}, err
	}
	if flags.HostedZoneKeepBase && !flags.HostedZoneDelete {
		return DestroyFlags{}, errors.New("--hosted-zone-keep-base must be used together with --hosted-zone-delete")
	}

	flags.SkipBaseTerraform = skipBaseTerraform

	return flags, nil

}
