/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// deployMetaphorCmd represents the deployMetaphor command
var deployMetaphorCmd = &cobra.Command{
	Use:   "deploy-metaphor",
	Short: "Add metaphor applications to the cluster",
	Long:  `TBD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("deployMetaphor called")
		config := configs.ReadConfig()
		globalFlags, err := flagset.ProcessGlobalFlags(cmd)
		if err != nil {
			return err
		}

		log.Printf("cloning and detokenizing the metaphor-template repository")
		prepareKubefirstTemplateRepo(config, viper.GetString("gitops.owner"), "metaphor", "", viper.GetString("template.tag"))
		log.Println("clone and detokenization of metaphor-template repository complete")

		log.Printf("cloning and detokenizing the metaphor-go-template repository")
		prepareKubefirstTemplateRepo(config, viper.GetString("gitops.owner"), "metaphor-go", "", viper.GetString("template.tag"))
		log.Println("clone and detokenization of metaphor-go-template repository complete")

		log.Printf("cloning and detokenizing the metaphor-frontend-template repository")
		prepareKubefirstTemplateRepo(config, viper.GetString("gitops.owner"), "metaphor-frontend", "", viper.GetString("template.tag"))
		log.Println("clone and detokenization of metaphor-frontend-template repository complete")

		if !viper.GetBool("gitlab.metaphor-pushed") {
			log.Println("Pushing metaphor repo to origin gitlab")
			gitlab.PushGitRepo(globalFlags.DryRun, config, "gitlab", "metaphor")
			viper.Set("gitlab.metaphor-pushed", true)
			viper.WriteConfig()
			log.Println("clone and detokenization of metaphor-frontend-template repository complete")
		}

		// Go template
		if !viper.GetBool("gitlab.metaphor-go-pushed") {
			log.Println("Pushing metaphor-go repo to origin gitlab")
			gitlab.PushGitRepo(globalFlags.DryRun, config, "gitlab", "metaphor-go")
			viper.Set("gitlab.metaphor-go-pushed", true)
			viper.WriteConfig()
		}

		// Frontend template
		if !viper.GetBool("gitlab.metaphor-frontend-pushed") {
			log.Println("Pushing metaphor-frontend repo to origin gitlab")
			gitlab.PushGitRepo(globalFlags.DryRun, config, "gitlab", "metaphor-frontend")
			viper.Set("gitlab.metaphor-frontend-pushed", true)
			viper.WriteConfig()
		}
		return nil
	},
}

func init() {
	actionCmd.AddCommand(deployMetaphorCmd)
	flagset.DefineGlobalFlags(deployMetaphorCmd)
}
