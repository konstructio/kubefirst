package flagset

import (
	"errors"
	"log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/addon"

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

// DefineGithubCmdFlags - define github flags
func DefineGithubCmdFlags(currentCommand *cobra.Command) {
	currentCommand.Flags().String("github-org", "", "Github Org of repos")
	currentCommand.Flags().String("github-owner", "", "Github owner of repos")
	currentCommand.Flags().String("github-host", "github.com", "Github URL")
	currentCommand.Flags().String("github-user", "", "Github user")

	err := viper.BindPFlag("github.org", currentCommand.Flags().Lookup("config.github-org"))
	if err != nil {
		log.Println("Error Binding flag: github.org")
	}
	err = viper.BindPFlag("github.host", currentCommand.Flags().Lookup("config.github-host"))
	if err != nil {
		log.Println("Error Binding flag: github.host")
	}
	err = viper.BindPFlag("github.user", currentCommand.Flags().Lookup("config.github-user"))
	if err != nil {
		log.Println("Error Binding flag: github.user")
	}

}

// ProcessGithubAddCmdFlags - Process github flags or vars
func ProcessGithubAddCmdFlags(cmd *cobra.Command) (GithubAddCmdFlags, error) {

	config := configs.ReadConfig()
	flags := GithubAddCmdFlags{}
	flags.GithubEnable = false
	user, err := ReadConfigString(cmd, "github-user")
	if err != nil {
		log.Println("Error Processing - github-user flag")
		return flags, err
	}
	org, err := ReadConfigString(cmd, "github-org")
	if err != nil {
		log.Println("Error Processing - github-org flag")
		return flags, err
	}

	// if GitHub installation, and GitHub personal access token is not provided, inform that the token is required for
	// GitHub installations
	if len(user) > 0 && len(org) > 0 && len(config.GitHubPersonalAccessToken) == 0 {
		errorMsg := "GITHUB_AUTH_TOKEN is required for GitHub installation"
		log.Println(errorMsg)
		return GithubAddCmdFlags{}, errors.New(errorMsg)
	}

	owner, err := ReadConfigString(cmd, "github-owner")
	if err != nil {
		log.Println("Error Processing - github-owner flag")
		return flags, err
	}

	host, err := ReadConfigString(cmd, "github-host")
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

	if flags.GithubEnable {
		addon.AddAddon("github")
	} else {
		addon.AddAddon("gitlab")
	}

	return flags, nil

}
