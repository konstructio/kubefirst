package cmd

import (
	"fmt"
	"log"

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

		// fmt.Println("cloning gitops-template repository")

		gitopsTemplateBranch := viper.GetString("template-repo.gitops.branch")
		gitopsTemplateURL := viper.GetString("template-repo.gitops.url")
		k1GitopsDir := viper.GetString("kubefirst.k1-directory-path")

		fmt.Println(gitopsTemplateBranch)
		fmt.Println(gitopsTemplateURL)
		fmt.Println(k1GitopsDir)

		repo, err := gitClient.CloneRefSetMain(gitopsTemplateBranch, k1GitopsDir, gitopsTemplateURL)
		if err != nil {
			log.Print("error opening repo at:", k1GitopsDir)
		}

		fmt.Println(repo)

		// todo clone metaphor-frontend and AdjustMetaphorContent
		// use previous logic to copy to ~/.k1/argo-workflows

		return nil
	},
}

func init() {
	rootCmd.AddCommand(actionCmd)
}
