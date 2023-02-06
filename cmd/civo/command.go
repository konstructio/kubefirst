package civo

import "github.com/spf13/cobra"

var (
	adminEmailFlag             string
	cloudRegionFlag            string
	clusterNameFlag            string
	clusterTypeFlag            string
	githubOwnerFlag            string
	gitopsTemplateURLFlag      string
	gitopsTemplateBranchFlag   string
	metaphorTemplateBranchFlag string
	metaphorTemplateURLFlag    string
	domainNameFlag             string
	kbotPasswordFlag           string
	silentModeFlag             bool
	useTelemetryFlag           bool
)

func NewCommand() *cobra.Command {

	civoCmd := &cobra.Command{
		Use:   "civo",
		Short: "kubefirst civo installation",
		Long:  "kubefirst civo",
	}

	// on error, doesnt show helper/usage
	civoCmd.SilenceUsage = true

	// wire up new commands
	civoCmd.AddCommand(Destroy())
	civoCmd.AddCommand(Create())

	return civoCmd
}

func Create() *cobra.Command {
	createCmd := &cobra.Command{
		Use:     "create",
		Short:   "create the k",
		Long:    "todo",
		PreRunE: validateCivo,
		RunE:    runCivo,
	}

	// todo review defaults and update descriptions
	createCmd.Flags().StringVar(&adminEmailFlag, "admin-email", "", "email address for let's encrypt certificate notifications")
	createCmd.Flags().StringVar(&cloudRegionFlag, "cloud-region", "NYC1", "the civo region to provision infrastructure in")
	createCmd.Flags().StringVar(&clusterNameFlag, "cluster-name", "kubefirst", "the name of the cluster to create")
	createCmd.Flags().StringVar(&clusterTypeFlag, "cluster-type", "mgmt", "the type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().StringVar(&domainNameFlag, "domain-name", "", "the Civo DNS Name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com)")
	createCmd.Flags().StringVar(&githubOwnerFlag, "github-owner", "", "the GitHub owner of the new gitops and metaphor repositories")
	createCmd.MarkFlagRequired("admin-email")
	createCmd.MarkFlagRequired("domain-name")
	createCmd.MarkFlagRequired("github-owner")
	createCmd.Flags().StringVar(&gitopsTemplateBranchFlag, "gitops-template-branch", "main", "the branch to clone for the gitops-template repository")
	createCmd.Flags().StringVar(&gitopsTemplateURLFlag, "gitops-template-url", "https://github.com/kubefirst/gitops-template.git", "the fully qualified url to the gitops-template repository to clone")
	createCmd.Flags().StringVar(&kbotPasswordFlag, "kbot-password", "", "the default password to use for the kbot user")
	createCmd.Flags().StringVar(&metaphorTemplateBranchFlag, "metaphor-template-branch", "main", "the branch to clone for the metaphor-template repository")
	createCmd.Flags().StringVar(&metaphorTemplateURLFlag, "metaphor-template-url", "https://github.com/kubefirst/metaphor-frontend-template.git", "the fully qualified url to the metaphor-template repository to clone")

	createCmd.Flags().BoolVar(&silentModeFlag, "silent-mode", false, "suppress output to the terminal")
	createCmd.Flags().BoolVar(&useTelemetryFlag, "use-telemetry", true, "whether to emit telemetry")
	return createCmd
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
