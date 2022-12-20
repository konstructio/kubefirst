/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/github"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// githubAddCmd represents the setupGithub command
var githubAddCmd = &cobra.Command{
	Use:   "add-github",
	Short: "Setup github for kubefirst install",
	Long:  `Prep github account to be used for Kubefirst installation `,
	RunE: func(cmd *cobra.Command, args []string) error {

		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}

		if globalFlags.DryRun {
			log.Info().Msg("[#99] Dry-run mode, githubAddCmd skipped.")
			return nil
		}

		log.Debug().Msgf("Org used: %s", viper.GetString("github.owner"))
		log.Debug().Msgf("dry-run: %s", globalFlags.DryRun)

		if !viper.GetBool("github.terraformapplied.gitops") {

			progressPrinter.IncrementTracker("step-github", 1)
			informUser("Creating gitops repository with terraform in GitHub", globalFlags.SilentMode)

			github.ApplyGitHubTerraform(globalFlags.DryRun)

			informUser("GitHub terraform applied", globalFlags.SilentMode)
			progressPrinter.IncrementTracker("step-github", 1)
		}

		log.Info().Msg("GitHub terraform Executed and uploaded ssh key to user with Success")
		return nil
	},
}

func init() {
	actionCmd.AddCommand(githubAddCmd)
	currentCommand := githubAddCmd
	flagset.DefineGlobalFlags(currentCommand)

}
