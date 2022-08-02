/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/spf13/cobra"
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
		//gitlab.PushGitRepo(dryRun, config, "gitlab", "metaphor")
		// make a github version of it

		err := githubAddCmd.RunE(cmd, args)
		if err != nil {
			return err
		}

		informUser("Created Github Repo - gitops/metaphor")

		//populate

		//gitlab.PushGitRepo(dryRun, config, "gitlab", "gitops")
		// make a github version of it
		informUser("Creating K8S Cluster")
		//terraform.ApplyBaseTerraform(dryRun, directory)

		// this should be handled by the process detokinize
		//!-New
		informUser("Point registry to github") // this should be handled by the process detokinize
		informUser("Add github runner")

		//!-Old
		informUser("Setup ArgoCD")
		informUser("Wait Vailt to be ready")
		informUser("Unseal Vault")
		informUser("Do we need terraform Github?")
		informUser("Setup Vault")
		informUser("Setup OICD - Github/Argo")
		informUser("Final Argo Synch")
		informUser("Wait ArgoCD to be ready")
		//sendCompleteInstallTelemetry(dryRun, useTelemetry)

		//!-New
		informUser("Show Hand-off screen")
		//reports.CreateHandOff
		//reports.CommandSummary(handOffData)
		time.Sleep(time.Millisecond * 2000)
		return nil
	},
}

func init() {
	clusterCmd.AddCommand(createGithubCmd)
	currentCommand := createGithubCmd
	currentCommand.Flags().String("github-org", "", "Github Org of repos")
	currentCommand.Flags().String("github-owner", "", "Github Owner of repos")
	currentCommand.Flags().String("github-host", "github.com", "Github repo, usally github.com, but it can change on enterprise customers.")
}
