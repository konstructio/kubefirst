/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"time"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/metaphor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// deployMetaphorCmd represents the deployMetaphor command
var deployMetaphorCmd = &cobra.Command{
	Use:   "deploy-metaphor",
	Short: "Add metaphor applications to the cluster",
	Long:  `TBD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info().Msg("deployMetaphor called")
		start := time.Now()
		defer func() {
			//The goal of this code is to track execution time
			duration := time.Since(start)
			log.Info().Msgf("[000] deploy-metaphor duration is %s", duration)

		}()

		if viper.GetBool("option.metaphor.skip") {
			log.Info().Msg("[99] Deployment of metpahor microservices skiped")
			return nil
		}

		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}
		/*
			config := configs.ReadConfig()
			repos := [3]string{"metaphor", "metaphor-go", "metaphor-frontend"}
			for _, repoName := range repos {
				directory := fmt.Sprintf("%s/%s", config.K1FolderPath, repoName)
				_ = os.RemoveAll(directory)
				log.Println("Removed repo pre-clone:", directory)
			}
		*/
		if viper.GetString("git-provider") == "github" {
			return metaphor.DeployMetaphorGithub(globalFlags)
		} else {
			return metaphor.DeployMetaphorGitlab(globalFlags)
		}

	},
}

func init() {
	actionCmd.AddCommand(deployMetaphorCmd)
	flagset.DefineGlobalFlags(deployMetaphorCmd)
}
