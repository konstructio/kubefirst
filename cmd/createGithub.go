/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/google/go-github/v45/github"
)

// createGithubCmd represents the createGithub command
var createGithubCmd = &cobra.Command{
	Use:   "create-github",
	Short: "create a kubefirst management cluster with github as Git Repo",
	Long:  `Create a kubefirst cluster using github as the Git Repo and setup integrations`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("createGithub called")
		progressPrinter.GetInstance()
		progressPrinter.SetupProgress(4)
		config := configs.ReadConfig()
		log.Printf(config.AwsProfile)
		infoCmd.Run(cmd, args)

		progressPrinter.AddTracker("step-0", "Test Installer ", 4)
		//sendStartedInstallTelemetry(dryRun, useTelemetry)
		informUser("Create Github Org")
		informUser("Create Github Repo - gitops")
		createRepo("gitops")
		//gitlab.PushGitRepo(dryRun, config, "gitlab", "metaphor")
		// make a github version of it
		informUser("Create Github Repo - metaphot")
		//gitlab.PushGitRepo(dryRun, config, "gitlab", "gitops")
		// make a github version of it
		informUser("Creating K8S Cluster")
		//terraform.ApplyBaseTerraform(dryRun, directory)
		informUser("Setup ArgoCD")
		informUser("Wait Vailt to be ready")
		informUser("Unseal Vault")
		informUser("Do we need terraform Github?")
		informUser("Setup Vault")
		informUser("Setup OICD - Github/Argo")
		informUser("Final Argo Synch")
		informUser("Wait ArgoCD to be ready")
		//sendCompleteInstallTelemetry(dryRun, useTelemetry)
		informUser("Show Hand-off screen")
		//reports.CreateHandOff
		//reports.CommandSummary(handOffData)
		time.Sleep(time.Millisecond * 2000)
		return nil
	},
}

func init() {
	clusterCmd.AddCommand(createGithubCmd)

}

func createRepo(name string) {
	token := os.Getenv("GITHUB_AUTH_TOKEN")
	if token == "" {
		log.Fatal("Unauthorized: No token present")
	}
	if name == "" {
		log.Fatal("No name: New repos must be given a name")
	}
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	isPrivate := true
	autoInit := true
	description := "sample"
	organization := os.Getenv("ORG")
	r := &github.Repository{Name: &name,
		Private:     &isPrivate,
		Description: &description,
		AutoInit:    &autoInit}
	repo, _, err := client.Repositories.Create(ctx, organization, r)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Successfully created new repo: %v\n", repo.GetName())
}
