package flagset

import (
	"log"

	"github.com/kubefirst/kubefirst/internal/addon"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GithubAddCmdFlags - Struct with flags used by githubAddCmd
type GithubAddCmdFlags struct {
	GithubOwner string
	GithubUser  string
	GithubHost  string
}

// DefineGithubCmdFlags - define github flags
func DefineGithubCmdFlags(currentCommand *cobra.Command) {
	currentCommand.Flags().String("github-host", "github.com", "Github URL")
	currentCommand.Flags().String("github-owner", "", "Github owner of repos")
	currentCommand.Flags().String("github-user", "", "Github user")

	err := viper.BindPFlag("github.host", currentCommand.Flags().Lookup("github-host"))
	if err != nil {
		log.Println("Error Binding flag: github.host")
	}

	err = viper.BindPFlag("github.owner", currentCommand.Flags().Lookup("github-owner"))
	if err != nil {
		log.Println("Error Binding flag: github.owner")
	}

	err = viper.BindPFlag("github.user", currentCommand.Flags().Lookup("github-user"))
	if err != nil {
		log.Println("Error Binding flag: github.user")
	}
}

// ProcessGithubAddCmdFlags - Process github flags or vars
func ProcessGithubAddCmdFlags(cmd *cobra.Command) (GithubAddCmdFlags, error) {

	flags := GithubAddCmdFlags{}
	user, err := ReadConfigString(cmd, "github-user")
	if err != nil {
		log.Println("Error Processing - github-user flag")
		return flags, err
	}
	if user == "" {
		user = viper.GetString("github.user")
	}

	owner, err := ReadConfigString(cmd, "github-owner")
	if err != nil {
		log.Println("Error Processing - github-owner flag")
		return flags, err
	}
	if owner == "" {
		owner = viper.GetString("github.owner")
	}

	host, err := ReadConfigString(cmd, "github-host")
	if err != nil {
		log.Println("Error Processing - github-host flag")
		return flags, err
	}

	flags.GithubHost = host
	flags.GithubOwner = owner
	flags.GithubUser = user

	viper.Set("github.host", flags.GithubHost)
	viper.Set("github.owner", flags.GithubOwner)
	viper.Set("github.user", flags.GithubUser)
	viper.WriteConfig()

	gitProvider, err := cmd.Flags().GetString("git-provider")
	if err != nil {
		log.Print(err)
	}
	log.Println(gitProvider)

	if gitProvider == "github" {
		addon.AddAddon("github")
	} else {
		addon.AddAddon("gitlab")
	}

	return flags, nil

}
