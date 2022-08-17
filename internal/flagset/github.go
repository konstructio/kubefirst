package flagset

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GithubAddCmdFlags - Struct with flags used by githubAddCmd
type GithubAddCmdFlags struct {
	GithubOwner  string
	GithubUser   string
	GithubOrg    string
	GithubHost   string
	GithubEnable bool
}

func DefineGithubCmdFlags(currentCommand *cobra.Command) {
	currentCommand.Flags().String("github-org", "", "Github Org of repos")
	currentCommand.Flags().String("github-owner", "", "Github owner of repos")
	currentCommand.Flags().String("github-host", "github.com", "Github URL")
	currentCommand.Flags().String("github-user", "", "Github user")

	viper.BindPFlag("github.org", currentCommand.Flags().Lookup("github-org"))
	viper.BindPFlag("github.host", currentCommand.Flags().Lookup("github-host"))
	viper.BindPFlag("github.user", currentCommand.Flags().Lookup("github-user"))

}

func ProcessGithubAddCmdFlags(cmd *cobra.Command) (GithubAddCmdFlags, error) {
	flags := GithubAddCmdFlags{}
	flags.GithubEnable = false
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
	if owner != "" {
		flags.GithubEnable = true
	}
	flags.GithubOwner = owner
	flags.GithubOrg = org
	flags.GithubUser = user
	viper.Set("github.owner", flags.GithubOwner)
	viper.Set("github.enabled", flags.GithubEnable)
	return flags, nil

}
