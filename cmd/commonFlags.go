package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func defineGlobalFlags(currentCommand *cobra.Command) {
	currentCommand.Flags().Bool("use-telemetry", true, "installer will not send telemetry about this installation")
	currentCommand.Flags().Bool("dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
}

func defineGithubCmdFlags(currentCommand *cobra.Command) {
	currentCommand.Flags().String("github-org", "", "Github Org of repos")
	currentCommand.Flags().String("github-owner", "", "Github owner of repos")
	currentCommand.Flags().String("github-host", "github.com", "Github URL")
	currentCommand.Flags().String("github-user", "", "Github user")

	viper.BindPFlag("github.host", githubAddCmd.Flags().Lookup("github-host"))
	viper.BindPFlag("github.org", githubAddCmd.Flags().Lookup("github-org"))
	viper.BindPFlag("github.owner", githubAddCmd.Flags().Lookup("github-owner"))
	viper.BindPFlag("github.owner", githubAddCmd.Flags().Lookup("github-owner"))

}

// GithubAddCmdFlags - Struct with flags used by githubAddCmd
type GithubAddCmdFlags struct {
	GithubOwner string
	GithubUser  string
	GithubOrg   string
	GithubHost  string
}

func processGithubAddCmdFlags(cmd *cobra.Command) (GithubAddCmdFlags, error) {
	flags := GithubAddCmdFlags{}
	user, err := cmd.Flags().GetString("github-user")
	if err != nil {
		log.Println("Error Processing - github-user flag")
		return flags, err
	}
	org, err := cmd.Flags().GetString("github-org")
	if err != nil {
		log.Println("Error Processing - github-org flag")
		return flags, err
	}

	owner, err := cmd.Flags().GetString("github-owner")
	if err != nil {
		log.Println("Error Processing - github-owner flag")
		return flags, err
	}

	host, err := cmd.Flags().GetString("github-host")
	if err != nil {
		log.Println("Error Processing - github-host flag")
		return flags, err
	}
	flags.GithubHost = host

	if owner == "" {
		if org == "" {
			owner = user
		} else {
			owner = org
		}

	}
	flags.GithubOwner = owner
	flags.GithubOrg = org
	flags.GithubUser = user
	return flags, nil

}

type GlobalFlags struct {
	DryRun       bool
	UseTelemetry bool
}

func processGlobalFlags(cmd *cobra.Command) (GlobalFlags, error) {
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
