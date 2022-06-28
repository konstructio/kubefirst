/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	gitlab "github.com/xanzy/go-gitlab"
)

// gitlabCmd represents the gitlab command
var gitlabCmd = &cobra.Command{
	Use:   "gitlab",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		git, err := gitlab.NewClient(
			viper.GetString("gitlab.token"),
			gitlab.WithBaseURL("https://gitlab.kube1st.com/api/v4"),
		)
		if err != nil {
			log.Fatal(err)
		}

		// Create an application
		opts := &gitlab.CreateApplicationOptions{
			Name:         gitlab.String("argo2"),
			Confidential: gitlab.Bool(false),
			RedirectURI:  gitlab.String("https://argocd.kubefirst.io/auth/callback2"),
			Scopes:       gitlab.String("api read_user"),
		}
		created, _, err := git.Applications.CreateApplication(opts)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Last created application : %v", created)

		// List all applications
		applications, _, err := git.Applications.ListApplications(&gitlab.ListApplicationsOptions{})
		if err != nil {
			log.Fatal(err)
		}

		for _, app := range applications {
			log.Printf("Found app : %v", app)
		}

	},
}

func init() {
	nebulousCmd.AddCommand(gitlabCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// gitlabCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// gitlabCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
