package flagset

import (
	"fmt"
	"log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// DefineInstallerGenericFlags - define installer  flags for CLI
type InstallerGenericFlags struct {
	ClusterName      string
	AdminEmail       string
	BotPassword      string
	Cloud            string
	OrgGitops        string
	BranchGitops     string //former: "version-gitops"
	BranchMetaphor   string
	RepoGitops       string //To support forks
	TemplateTag      string //To support forks
	SkipMetaphor     bool
	ExperimentalMode bool
}

func DefineInstallerGenericFlags(currentCommand *cobra.Command) {
	// Generic Installer flags:
	currentCommand.Flags().String("cluster-name", "kubefirst", "the cluster name, used to identify resources on cloud provider")
	currentCommand.Flags().String("admin-email", "", "the email address for the administrator as well as for lets-encrypt certificate emails")
	currentCommand.Flags().String("cloud", "", "the cloud to provision infrastructure in")
	currentCommand.Flags().String("gitops-owner", "kubefirst", "git owner of gitops, this may be a user or a org to support forks for testing")
	currentCommand.Flags().String("gitops-repo", "gitops", "version/branch used on git clone")
	currentCommand.Flags().String("gitops-branch", "", "version/branch used on git clone - former: version-gitops flag")
	currentCommand.Flags().String("metaphor-branch", "", "version/branch used on git clone - former: version-gitops flag")
	currentCommand.Flags().String("bot-password", "", "initial password to use while establishing the bot account")
	currentCommand.Flags().String("template-tag", configs.K1Version, `fallback tag used on git clone.
  Details: if "gitops-branch" is provided, branch("gitops-branch") has precedence and installer will attempt to clone branch("gitops-branch") first,
  if it fails, then fallback it will attempt to clone the tag provided at "template-tag" flag`)
	currentCommand.Flags().Bool("skip-metaphor-services", false, "whether to skip the deployment of metaphor micro-services demo applications")
	currentCommand.Flags().Bool("experimental-mode", false, `whether to allow experimental behavior or developer mode of installer, 
  not recommended for most use cases, as it may mix versions and create unexpected behavior.`)
}

// ProcessInstallerGenericFlags - Read values of CLI parameters for installer flags
func ProcessInstallerGenericFlags(cmd *cobra.Command) (InstallerGenericFlags, error) {
	flags := InstallerGenericFlags{}
	defer func() {
		err := viper.WriteConfig()
		if err != nil {
			log.Println(err)
		}
	}()

	adminEmail, err := ReadConfigString(cmd, "admin-email")
	if err != nil {
		return InstallerGenericFlags{}, err
	}
	flags.AdminEmail = adminEmail
	log.Println("adminEmail:", adminEmail)
	viper.Set("adminemail", adminEmail)

	clusterName, err := ReadConfigString(cmd, "cluster-name")
	if err != nil {
		return InstallerGenericFlags{}, err
	}
	viper.Set("cluster-name", clusterName)
	log.Println("cluster-name:", clusterName)
	flags.ClusterName = clusterName

	cloud, err := ReadConfigString(cmd, "cloud")
	if err != nil {
		return InstallerGenericFlags{}, err
	}
	viper.Set("cloud", cloud)
	log.Println("cloud:", cloud)
	flags.Cloud = cloud

	branchGitOps, err := ReadConfigString(cmd, "gitops-branch")
	if err != nil {
		return InstallerGenericFlags{}, err
	}
	viper.Set("gitops.branch", branchGitOps)
	log.Println("gitops.branch:", branchGitOps)
	flags.BranchGitops = branchGitOps

	botPassword, err := ReadConfigString(cmd, "bot-password")
	if err != nil {
		return InstallerGenericFlags{}, err
	}
	viper.Set("botpassword", botPassword)
	flags.BotPassword = botPassword

	metaphorGitOps, err := ReadConfigString(cmd, "metaphor-branch")
	if err != nil {
		return InstallerGenericFlags{}, err
	}
	viper.Set("metaphor.branch", metaphorGitOps)
	log.Println("metaphor.branch:", metaphorGitOps)
	flags.BranchMetaphor = metaphorGitOps

	repoGitOps, err := ReadConfigString(cmd, "gitops-repo")
	if err != nil {
		return InstallerGenericFlags{}, err
	}
	viper.Set("gitops.repo", repoGitOps)
	log.Println("gitops.repo:", repoGitOps)
	flags.RepoGitops = repoGitOps

	ownerGitOps, err := ReadConfigString(cmd, "gitops-owner")
	if err != nil {
		return InstallerGenericFlags{}, err
	}
	viper.Set("gitops.owner", ownerGitOps)
	log.Println("gitops.owner:", ownerGitOps)
	flags.RepoGitops = ownerGitOps

	templateTag, err := ReadConfigString(cmd, "template-tag")
	if err != nil {
		return InstallerGenericFlags{}, err
	}
	viper.Set("template.tag", templateTag)
	log.Println("template.tag", templateTag)
	flags.TemplateTag = templateTag

	skipMetaphor, err := ReadConfigBool(cmd, "skip-metaphor-services")
	if err != nil {
		log.Println("Error processing skip-metaphor-services:", err)
		return InstallerGenericFlags{}, err
	}
	viper.Set("option.metaphor.skip", skipMetaphor)
	log.Println("option.metaphor.skip", skipMetaphor)
	flags.SkipMetaphor = skipMetaphor

	experimentalMode, err := ReadConfigBool(cmd, "experimental-mode")
	if err != nil {
		log.Println("Error processing experimental-mode:", err)
		return InstallerGenericFlags{}, err
	}
	viper.Set("option.kubefirst.experimental", experimentalMode)
	log.Println("option.kubefirst.experimental", experimentalMode)
	flags.ExperimentalMode = experimentalMode

	if viper.GetBool("github.enabled") && flags.BotPassword == "" {
		return InstallerGenericFlags{}, fmt.Errorf("must provide bot-password argument for github installations of kubefirst")

	}

	return experimentalModeTweaks(flags), nil
}

func experimentalModeTweaks(flags InstallerGenericFlags) InstallerGenericFlags {
	//Handling the scenario there is no fallback tag, in development mode.
	if flags.ExperimentalMode && configs.K1Version == "" && flags.BranchGitops == "" {
		//no branch or tag will be set, failing action of cloning templates.
		//forcing main as branch
		flags.BranchGitops = "main"
		log.Println("[W1] Warning: Fallback mechanism was disabled due to the use of experimental mode, be sure this was the intented action.")
		log.Println("[W1] Warning: IF you are development mode, please check documentation on how to do this via LDFLAGS to avoid unexpected actions")
		viper.Set("gitops.branch", flags.BranchGitops)
		log.Println("[W1]  Warning: Overrride gitops.branch:", flags.BranchGitops)

	}
	if flags.ExperimentalMode && configs.K1Version == "" && flags.BranchMetaphor == "" {
		//no branch or tag will be set, failing action of cloning templates.
		//forcing main as branch
		flags.BranchMetaphor = "main"
		log.Println("[W1] Warning: Fallback mechanism was disabled due to the use of experimental mode, be sure this was the intented action.")
		log.Println("[W1] Warning: IF you are development mode, please check documentation on how to do this via LDFLAGS to avoid unexpected actions")
		viper.Set("metaphor.branch", flags.BranchMetaphor)
		log.Println("[W1]  Warning: Overrride metaphor.branch:", flags.BranchMetaphor)

	}
	return flags
}
