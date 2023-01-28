package cmd

import (
	"fmt"

	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
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

		//! todo

		gitopsTemplateBranch := viper.GetString("template-repo.gitops.branch")
		gitopsTemplateURL := viper.GetString("template-repo.gitops.url")
		cloudProvider := viper.GetString("cloud-provider")
		gitProvider := viper.GetString("git-provider")
		k1GitopsDir := viper.GetString("kubefirst.k1-gitops-dir")
		k1Dir := viper.GetString("kubefirst.k1-directory-path")
		clusterName := viper.GetString("kubefirst.cluster-name")
		clusterType := viper.GetString("kubefirst.cluster-type")
		destinationGitopsRepoURL := viper.GetString("github.repo.gitops.giturl")

		fmt.Println("gitopsTemplateBranch: ", gitopsTemplateBranch)
		fmt.Println("gitopsTemplateURL: ", gitopsTemplateURL)
		fmt.Println("cloudProvider: ", cloudProvider)
		fmt.Println("gitProvider: ", gitProvider)
		fmt.Println("k1GitopsDir: ", k1GitopsDir)

		gitopsRepo, err := gitClient.CloneBranchSetMain(gitopsTemplateBranch, gitopsTemplateURL, k1GitopsDir)
		if err != nil {
			log.Print("error opening repo at:", k1GitopsDir)
		}

		log.Info().Msg("gitops repository clone complete")

		pkg.AdjustGitopsTemplateContent(cloudProvider, clusterName, clusterType, gitProvider, k1Dir, k1GitopsDir)

		pkg.DetokenizeCivoGithubGitops(k1GitopsDir)

		gitClient.AddRemote(destinationGitopsRepoURL, gitProvider, gitopsRepo)

		gitClient.Commit(gitopsRepo, "committing initial detokenized gitops-template repo content")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(actionCmd)
}
