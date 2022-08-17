package flagset

import (
	"log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func DefineInstallerGenericFlags(currentCommand *cobra.Command) {
	//Gewneric Installer flags:
	config := configs.ReadConfig()
	currentCommand.Flags().String("cluster-name", "kubefirst", "the cluster name, used to identify resources on cloud provider")
	currentCommand.Flags().String("admin-email", "", "the email address for the administrator as well as for lets-encrypt certificate emails")
	currentCommand.Flags().String("cloud", "", "the cloud to provision infrastructure in")
	currentCommand.Flags().String("repo-gitops", "https://github.com/kubefirst/gitops-template-gh.git", "version/branch used on git clone")
	currentCommand.Flags().String("branch-gitops", "", "version/branch used on git clone - former: version-gitops flag")
	currentCommand.Flags().String("template-tag", config.KubefirstVersion, `fallback tag used on git clone.
  Details: if "branch-gitops" is provided, branch("branch-gitops") has precedence and installer will attempt to clone branch("branch-gitops") first,
  if it fails, then fallback it will attempt to clone the tag provided at "template-tag" flag`)
}

type InstallerGenericFlags struct {
	ClusterName  string
	AdminEmail   string
	Cloud        string
	BranchGitops string //former: "version-gitops"
	RepoGitops   string //To support forks
	TemplateTag  string //To support forks
}

func ProcessInstallerGenericFlags(cmd *cobra.Command) (InstallerGenericFlags, error) {
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

	templateTag, err := cmd.Flags().GetString("template-tag")
	if err != nil {
		return flags, err
	}
	viper.Set("template.tag", templateTag)
	log.Println("template.tag", templateTag)
	flags.TemplateTag = templateTag

	return flags, nil
}
