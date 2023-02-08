package flagset

import (
	"errors"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/pkg"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/addon"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// DefineInstallerGenericFlags - define installer  flags for CLI
type InstallerGenericFlags struct {
	ClusterName      string
	AdminEmail       string
	BotPassword      string
	Cloud            string
	GitProvider      string
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
	currentCommand.Flags().String("cloud", "k3d", "the cloud to provision infrastructure in")
	currentCommand.Flags().String("git-provider", "github", "specify \"github\" or \"gitlab\" git provider. defaults to github.")
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
	currentCommand.Flags().StringSlice("addons", nil, `the name of addon to enable on create cluster:
  --addon foo or --addon foo,bar for example`)
}

// ProcessInstallerGenericFlags - Read values of CLI parameters for installer flags
func ProcessInstallerGenericFlags(cmd *cobra.Command) (InstallerGenericFlags, error) {
	flags := InstallerGenericFlags{}
	defer func() {
		err := viper.WriteConfig()
		if err != nil {
			log.Warn().Msgf("%s", err)
		}
	}()

	gitProvider, err := ReadConfigString(cmd, "git-provider")
	if err != nil {
		return InstallerGenericFlags{}, err
	}
	flags.GitProvider = gitProvider
	log.Info().Msgf("git provider: %s", gitProvider)
	viper.Set("git-provider", gitProvider)

	adminEmail, err := ReadConfigString(cmd, "admin-email")
	if err != nil {
		return InstallerGenericFlags{}, err
	}
	flags.AdminEmail = adminEmail
	log.Info().Msgf("adminEmail: %s", adminEmail)
	viper.Set("adminemail", adminEmail)

	clusterName, err := ReadConfigString(cmd, "cluster-name")
	if err != nil {
		return InstallerGenericFlags{}, err
	}
	viper.Set("cluster-name", clusterName)
	log.Info().Msgf("cluster-name: %s", clusterName)
	flags.ClusterName = clusterName

	cloud, err := ReadConfigString(cmd, "cloud")
	if err != nil {
		return InstallerGenericFlags{}, err
	}
	viper.Set("cloud", cloud)
	log.Info().Msgf("cloud: %s", cloud)
	flags.Cloud = cloud

	branchGitOps, err := ReadConfigString(cmd, "gitops-branch")
	if err != nil {
		return InstallerGenericFlags{}, err
	}
	viper.Set("gitops.branch", branchGitOps)
	log.Info().Msgf("gitops.branch: %s", branchGitOps)
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
	log.Info().Msgf("metaphor.branch: %s", metaphorGitOps)
	flags.BranchMetaphor = metaphorGitOps

	repoGitOps, err := ReadConfigString(cmd, "gitops-repo")
	if err != nil {
		return InstallerGenericFlags{}, err
	}
	viper.Set("gitops.repo", repoGitOps)
	log.Info().Msgf("gitops.repo: %s", repoGitOps)
	flags.RepoGitops = repoGitOps

	ownerGitOps, err := ReadConfigString(cmd, "gitops-owner")
	if err != nil {
		return InstallerGenericFlags{}, err
	}
	viper.Set("gitops.owner", ownerGitOps)
	log.Info().Msgf("gitops.owner: %s", ownerGitOps)
	flags.RepoGitops = ownerGitOps

	templateTag, err := ReadConfigString(cmd, "template-tag")
	if err != nil {
		return InstallerGenericFlags{}, err
	}
	viper.Set("template.tag", templateTag)
	log.Info().Msgf("template.tag: %s", templateTag)
	flags.TemplateTag = templateTag

	skipMetaphor, err := ReadConfigBool(cmd, "skip-metaphor-services")
	if err != nil {
		log.Warn().Msgf("Error processing skip-metaphor-services: %s", err)
		return InstallerGenericFlags{}, err
	}
	viper.Set("option.metaphor.skip", skipMetaphor)
	log.Info().Msgf("option.metaphor.skip: %t", skipMetaphor)
	flags.SkipMetaphor = skipMetaphor

	addonsFlag, err := ReadConfigStringSlice(cmd, "addons")
	if err != nil {
		log.Warn().Msgf("Error processing addons: %s", err)
		return InstallerGenericFlags{}, err
	}
	for _, s := range addonsFlag {
		addon.AddAddon(s)
	}
	//TODO: add unit test for this, after Thiago PR is merged on new append checks
	if flags.Cloud == pkg.CloudAws {
		//Adds mandatory addon for non-local install
		addon.AddAddon("cloud")
	}
	if flags.Cloud == pkg.CloudK3d {
		//Adds mandatory addon for local install
		addon.AddAddon("k3d")
	}

	experimentalMode, err := ReadConfigBool(cmd, "experimental-mode")
	if err != nil {
		log.Warn().Msgf("Error processing experimental-mode: %s", err)
		return InstallerGenericFlags{}, err
	}
	viper.Set("option.kubefirst.experimental", experimentalMode)
	log.Info().Msgf("option.kubefirst.experimental: %t", experimentalMode)
	flags.ExperimentalMode = experimentalMode

	err = validateInstallationFlags()
	if err != nil {
		log.Warn().Msgf("Error validateInstallationFlags: %s", err)
		return InstallerGenericFlags{}, err
	}

	return experimentalModeTweaks(flags), nil
}

func experimentalModeTweaks(flags InstallerGenericFlags) InstallerGenericFlags {
	//Handling the scenario there is no fallback tag, in development mode.
	if flags.ExperimentalMode && configs.K1Version == "" && flags.BranchGitops == "" {
		//no branch or tag will be set, failing action of cloning templates.
		//forcing main as branch
		flags.BranchGitops = "main"
		log.Warn().Msg("[W1] Warning: Fallback mechanism was disabled due to the use of experimental mode, be sure this was the intented action.")
		log.Warn().Msg("[W1] Warning: IF you are development mode, please check documentation on how to do this via LDFLAGS to avoid unexpected actions")
		viper.Set("gitops.branch", flags.BranchGitops)
		log.Warn().Msgf("[W1]  Warning: Overrride gitops.branch: %s", flags.BranchGitops)

	}
	if flags.ExperimentalMode && configs.K1Version == "" && flags.BranchMetaphor == "" {
		//no branch or tag will be set, failing action of cloning templates.
		//forcing main as branch
		flags.BranchMetaphor = "main"
		log.Warn().Msg("[W1] Warning: Fallback mechanism was disabled due to the use of experimental mode, be sure this was the intented action.")
		log.Warn().Msg("[W1] Warning: IF you are development mode, please check documentation on how to do this via LDFLAGS to avoid unexpected actions")
		viper.Set("metaphor.branch", flags.BranchMetaphor)
		log.Warn().Msgf("[W1]  Warning: Overrride metaphor.branch: %s", flags.BranchMetaphor)

	}
	return flags
}

// validateInstallationFlags: Validate installation major flags
func validateInstallationFlags() error {
	//If you are changind this rules, please ensure to update:
	// internal/flagset/init_test.go
	// todo validate on email address if not local
	// if len(viper.GetString("adminemail")) < 1 {
	// 	message := "missing flag --admin-email"
	// 	log.Println(message)
	// 	return errors.New(message)
	// }
	if len(viper.GetString("cloud")) < 1 {
		message := "missing flag --cloud, supported values: " + pkg.CloudAws + ", " + pkg.CloudK3d
		log.Warn().Msgf("%s", message)
		return errors.New(message)
	}

	if (viper.GetString("botpassword") != "") && len(viper.GetString("botpassword")) < 8 && (viper.GetString("git-provider") == "gitlab") {
		msg := "BotPassword (to GitLab flavor) is too short (minimum is 8 characters)"
		return errors.New(msg)
	}

	return nil
}
