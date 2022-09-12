/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"log"

	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// loadTemplateCmd represents the loadTemplate command
var loadTemplateCmd = &cobra.Command{
	Use:   "load-template",
	Short: "Clone and Detonize template repos",
	Long:  `TBD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Println("loadTemplate called")
		if viper.GetBool("github.repo.loaded") {
			log.Println("github.repo.loaded already executed, skiped")
			return nil
		}
		gitopsDir, err := gitClient.CloneRepoAndDetokenizeTemplate(viper.GetString("gitops.owner"), viper.GetString("gitops.repo"), "gitops", viper.GetString("gitops.branch"), viper.GetString("template.tag"))
		if err != nil {
			log.Printf("Error clonning and detokizing repo %s", "gitops")
			return err
		}
		log.Println("loadTemplate executed with success:", gitopsDir)
		viper.Set("github.repo.loaded", true)
		viper.WriteConfig()
		return nil

	},
}

func init() {
	actionCmd.AddCommand(loadTemplateCmd)
}
