package flagset

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GithubAddCmdFlags - Struct with flags used by githubAddCmd
type GithubAddCmdFlags struct {
	GithubOwner string
	GithubUser  string
	GithubOrg   string
	GithubHost  string
}

func DefineGithubCmdFlags(currentCommand *cobra.Command) {
	currentCommand.Flags().String("github-org", "", "Github Org of repos")
	currentCommand.Flags().String("github-owner", "", "Github owner of repos")
	currentCommand.Flags().String("github-host", "github.com", "Github URL")
	currentCommand.Flags().String("github-user", "", "Github user")

	viper.BindPFlag("github.host", currentCommand.Flags().Lookup("github-host"))
	viper.BindPFlag("github.org", currentCommand.Flags().Lookup("github-org"))
	viper.BindPFlag("github.owner", currentCommand.Flags().Lookup("github-owner"))
	viper.BindPFlag("github.owner", currentCommand.Flags().Lookup("github-owner"))

}

func ProcessGithubAddCmdFlags(cmd *cobra.Command) (GithubAddCmdFlags, error) {
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
