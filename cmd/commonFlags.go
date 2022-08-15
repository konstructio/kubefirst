package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func defineInstallerGenericFlags(currentCommand *cobra.Command) {
	//Gewneric Installer flags:
	currentCommand.Flags().String("cluster-name", "kubefirst", "the cluster name, used to identify resources on cloud provider")
	currentCommand.Flags().String("admin-email", "", "the email address for the administrator as well as for lets-encrypt certificate emails")
	currentCommand.MarkFlagRequired("admin-email")
	currentCommand.Flags().String("cloud", "", "the cloud to provision infrastructure in")
	currentCommand.MarkFlagRequired("cloud")
	currentCommand.Flags().String("version-gitops", "main", "version/branch used on git clone")
	currentCommand.Flags().String("repo-gitops", "https://github.com/kubefirst/gitops-template-gh.git", "version/branch used on git clone")
}

type InstallerGenericFlags struct {
	ClusterName  string
	AdminEmail   string
	Cloud        string
	BranchGitops string //former: "version-gitops"
	RepoGitops   string //To support forks
}

func processInstallerGenericFlags(cmd *cobra.Command) (InstallerGenericFlags, error) {
	flags := InstallerGenericFlags{}
	defer viper.WriteConfig()

	adminEmail, err := cmd.Flags().GetString("admin-email")
	if err != nil {
		return flags, err
	}
	flags.AdminEmail = adminEmail
	log.Println("adminEmail:", adminEmail)
	viper.Set("adminemail", adminEmail)

	clusterName, err := cmd.Flags().GetString("cluster-name")
	if err != nil {
		return flags, err
	}
	viper.Set("cluster-name", clusterName)
	log.Println("cluster-name:", clusterName)
	flags.ClusterName = clusterName

	cloud, err := cmd.Flags().GetString("cloud")
	if err != nil {
		return flags, err
	}
	viper.Set("cloud", cloud)
	log.Println("cloud:", cloud)
	flags.Cloud = cloud

	branchGitOps, err := cmd.Flags().GetString("branch-gitops")
	if err != nil {
		return flags, err
	}
	viper.Set("branch-gitops", branchGitOps)
	log.Println("branch-gitops:", branchGitOps)
	flags.BranchGitops = branchGitOps

	repoGitOps, err := cmd.Flags().GetString("repo-gitops")
	if err != nil {
		return flags, err
	}
	viper.Set("repo-gitops", repoGitOps)
	log.Println("repo-gitops:", repoGitOps)
	flags.RepoGitops = branchGitOps

	return flags, nil
}

// GithubAddCmdFlags - Struct with flags used by githubAddCmd
type GithubAddCmdFlags struct {
	GithubOwner string
	GithubUser  string
	GithubOrg   string
	GithubHost  string
}

func defineGithubCmdFlags(currentCommand *cobra.Command) {
	currentCommand.Flags().String("github-org", "", "Github Org of repos")
	currentCommand.Flags().String("github-owner", "", "Github owner of repos")
	currentCommand.Flags().String("github-host", "github.com", "Github URL")
	currentCommand.Flags().String("github-user", "", "Github user")

	viper.BindPFlag("github.host", githubAddCmd.Flags().Lookup("github-host"))
	viper.BindPFlag("github.org", githubAddCmd.Flags().Lookup("github-org"))
	viper.BindPFlag("github.owner", githubAddCmd.Flags().Lookup("github-owner"))
	viper.BindPFlag("github.owner", githubAddCmd.Flags().Lookup("github-owner"))

}

func processGithubAddCmdFlags(cmd *cobra.Command) (GithubAddCmdFlags, error) {
	flags := GithubAddCmdFlags{}
	user, err := cmd.Flags().GetString("github-user")
	if err != nil {
		log.Println("Error Processing - github-user flag")
		return flags, err
	}
	org, err := cmd.Flags().GetString("github-org")
	if err != nil {
		log.Println("Error Processing - github-org flag")
		return flags, err
	}

	owner, err := cmd.Flags().GetString("github-owner")
	if err != nil {
		log.Println("Error Processing - github-owner flag")
		return flags, err
	}

	host, err := cmd.Flags().GetString("github-host")
	if err != nil {
		log.Println("Error Processing - github-host flag")
		return flags, err
	}
	flags.GithubHost = host

	if owner == "" {
		if org == "" {
			owner = user
		} else {
			owner = org
		}

	}
	flags.GithubOwner = owner
	flags.GithubOrg = org
	flags.GithubUser = user
	return flags, nil

}
