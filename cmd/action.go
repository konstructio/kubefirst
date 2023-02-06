package cmd

import (
	"fmt"
	"log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// actionCmd represents the action command
var actionCmd = &cobra.Command{
	Use:   "action",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		k1MetaphorDir := viper.GetString("kubefirst.k1-metaphor-dir")
		gitopsTemplateURL := viper.GetString("template-repo.metaphor-frontend.url")
		gitopsTemplateBranch := "main"

		fmt.Println(configs.K1Version)

		if configs.K1Version != "" && configs.K1Version != "development" {
			gitopsTemplateBranch = configs.K1Version
		}
		fmt.Println("hello config.K1Version", configs.K1Version)
		_, err := gitClient.CloneRefSetMain(gitopsTemplateBranch, k1MetaphorDir, gitopsTemplateURL)
		if err != nil {
			log.Print("error opening repo at:", k1MetaphorDir)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(actionCmd)
}
