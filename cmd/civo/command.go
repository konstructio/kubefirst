package civo

import "github.com/spf13/cobra"

var (
	adminEmailFlag           string
	civoProfileFlag          string
	civoRegionFlag           string
	cloudProviderFlag        string
	civoClusterNameFlag      string
	githubOwner              string
	gitopsTemplateUrlFlag    string
	gitopsTemplateBranchFlag string
	gitProviderFlag          string
	civoDnsFlag              string
	kbotPasswordFlag         string
	silentModeFlag           bool
	useTelemetryFlag         bool
)

func NewCommand() *cobra.Command {

	civoCmd := &cobra.Command{
		Use:     "civo",
		Short:   "kubefirst civo installation",
		Long:    "kubefirst civo",
		PreRunE: validateCivo, // todo what should this function be called?
		RunE:    runCivo,
		// PostRunE: runPostCivo,
	}

	// todo review defaults and update descriptions
	civoCmd.Flags().StringVar(&adminEmailFlag, "admin-email", "jared@kubeshop.io", "email address for let's encrypt certificate notifications")
	civoCmd.Flags().StringVar(&civoDnsFlag, "dns", "k-ray.space", "the Civo DNS Name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com)")
	civoCmd.Flags().StringVar(&civoRegionFlag, "region", "NYC1", "the civo region to provision infrastructure in")
	civoCmd.Flags().StringVar(&kbotPasswordFlag, "kbot-password", "password", "the default password to use for the kbot user")
	civoCmd.Flags().StringVar(&cloudProviderFlag, "cloud-provider", "civo", "the git provider to use. (i.e. gitlab|github)")
	civoCmd.Flags().StringVar(&civoClusterNameFlag, "cluster-name", "kubefirst", "the name of the cluster to create")
	civoCmd.Flags().StringVar(&githubOwner, "github-owner", "your-dns-io", "the GitHub owner of the new gitops and metaphor repositories")
	// civoCmd.MarkFlagRequired("github-owner")
	civoCmd.Flags().StringVar(&gitopsTemplateBranchFlag, "gitops-template-branch", "civo-domain-refactor", "the branch to clone for the gitops-template repository")
	civoCmd.Flags().StringVar(&gitopsTemplateUrlFlag, "gitops-template-url", "https://github.com/kubefirst/gitops-template.git", "the fully qualified url to the gitops-template repository to clone")
	civoCmd.Flags().StringVar(&gitProviderFlag, "git-provider", "github", "the git provider to use. (i.e. gitlab|github)")

	civoCmd.Flags().BoolVar(&silentModeFlag, "silent-mode", false, "suppress output to the terminal")
	civoCmd.Flags().BoolVar(&useTelemetryFlag, "use-telemetry", true, "whether to emit telemetry")

	// on error, doesnt show helper/usage
	civoCmd.SilenceUsage = true

	// wire up new commands
	civoCmd.AddCommand(Destroy())

	return civoCmd
}

func Destroy() *cobra.Command {
	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "destroy civo cloud",
		Long:  "todo",
		RunE:  destroyCivo,
	}

	return destroyCmd
}
