/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// githubAddCmd represents the setupGithub command
var githubAddCmd = &cobra.Command{
	Use:   "add-github",
	Short: "Setup github for kubefirst install",
	Long:  `TBD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("githubAddCmd called")
		config := configs.ReadConfig()
		owner, err := cmd.Flags().GetString("github-owner")
		if err != nil {
			return err
		}
		viper.Set("github.owner", owner)
		viper.WriteConfig()

		org, err := cmd.Flags().GetString("github-org")
		if err != nil {
			return err
		}
		log.Println("Org used:", org)
		dryrun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			return err
		}
		log.Println("dry-run:", dryrun)

		if viper.GetBool("github.repo.added") {
			log.Println("github.repo.added already executed, skiped")
			return nil
		}
		if dryrun {
			log.Printf("[#99] Dry-run mode, githubAddCmd skipped.")
			return nil
		}

		gitWrapper := githubWrapper.New()
		gitWrapper.CreatePrivateRepo(org, "gitops", "Kubefirst Gitops")
		gitWrapper.CreatePrivateRepo(org, "metaphor", "Sample Kubefirst App")

		//Add Github SSHPublic key
		if viper.GetString("botPublicKey") != "" {
			key, err := gitWrapper.AddSSHKey("kubefirst-bot", viper.GetString("botPublicKey"))
			viper.Set("github.ssh.keyId", key.GetID())
			if err != nil {
				return err
			}
		}

		_, err = gitClient.CloneRepoAndDetokenize(config.GitopsTemplateURL, "gitops", "main")
		if err != nil {
			return err
		}
		_, err = gitClient.CloneRepoAndDetokenize(config.MetaphorTemplateURL, "metaphor", "main")
		if err != nil {
			return err
		}
		viper.Set("github.enabled", true)
		viper.Set("github.repo.added", true)
		viper.WriteConfig()
		return nil
	},
}

func init() {
	actionCmd.AddCommand(githubAddCmd)
	currentCommand := githubAddCmd
	currentCommand.Flags().Bool("dry-run", false, "set to dry-run mode, no changes done on cloud provider selected")
	currentCommand.Flags().String("github-org", "", "Github Org of repos")
	viper.BindPFlag("github.org", githubAddCmd.Flags().Lookup("github-org"))

}
