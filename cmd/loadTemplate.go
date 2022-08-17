/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/spf13/cobra"
)

// loadTemplateCmd represents the loadTemplate command
var loadTemplateCmd = &cobra.Command{
	Use:   "load-template",
	Short: "Clone and Detonize template repos",
	Long:  `TBD`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Println("loadTemplate called")
		config := configs.ReadConfig()
		_, err := gitClient.CloneRepoAndDetokenize(config.GitopsTemplateURL, "gitops", "main")
		if err != nil {
			log.Printf("Error clonning and detokizing repo %s", "gitops")
			return err
		}
		_, err = gitClient.CloneRepoAndDetokenize(config.MetaphorTemplateURL, "metaphor", "main")
		if err != nil {
			log.Printf("Error clonning and detokizing repo %s", "metaphor")
			return err
		}
		log.Println("loadTemplate executed with success")
		return nil

	},
}

func init() {
	actionCmd.AddCommand(loadTemplateCmd)

}
