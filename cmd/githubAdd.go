/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"log"

	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// githubAddCmd represents the setupGithub command
var githubAddCmd = &cobra.Command{
	Use:   "add-github",
	Short: "Setup github for kubefirst install",
	Long:  `Prepate github account to be used for Kubefirst installation `,
	RunE: func(cmd *cobra.Command, args []string) error {

		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}

		log.Println("Org used:", viper.GetString("github.org"))
		log.Println("dry-run:", globalFlags.DryRun)
		viper.Set("github.owner", viper.GetString("github.owner"))
		viper.WriteConfig()

		if viper.GetBool("github.repo.added") {
			log.Println("github.repo.added already executed, skiped")
			return nil
		}
		if globalFlags.DryRun {
			log.Printf("[#99] Dry-run mode, githubAddCmd skipped.")
			return nil
		}
		gitWrapper := githubWrapper.New()
		gitWrapper.CreatePrivateRepo(viper.GetString("github.org"), "gitops", "Kubefirst Gitops")

		//Add Github SSHPublic key
		if viper.GetString("botPublicKey") != "" {
			key, err := gitWrapper.AddSSHKey("kubefirst-bot", viper.GetString("botPublicKey"))
			if err != nil {
				log.Printf("Error Adding SSH key to github account")
				return err
			}
			viper.Set("github.ssh.keyId", key.GetID())
		} else {
			log.Printf("Missing key `botPublicKey` to be added on the account, step skipped.")
		}

		viper.Set("github.repo.added", true)
		viper.WriteConfig()
		log.Printf("github.repo.added - Executed with Success")
		return nil
	},
}

func init() {
	actionCmd.AddCommand(githubAddCmd)
	currentCommand := githubAddCmd
	flagset.DefineGlobalFlags(currentCommand)

}
