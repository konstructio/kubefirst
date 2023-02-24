package aws

import (
	"github.com/spf13/cobra"
)

var (
	// Create
	alertsEmailFlag            string
	cloudRegionFlag            string
	clusterNameFlag            string
	clusterTypeFlag            string
	domainNameFlag             string
	dryRun                     bool
	githubOwnerFlag            string
	gitopsTemplateBranchFlag   string
	gitopsTemplateURLFlag      string
	kbotPasswordFlag           string
	metaphorTemplateBranchFlag string
	metaphorTemplateURLFlag    string
	useTelemetryFlag           bool

	// Quota
	quotaShowAllFlag bool
)

func NewCommand() *cobra.Command {

	awsCmd := &cobra.Command{
		Use:   "aws",
		Short: "kubefirst aws installation",
		Long:  "kubefirst aws",
	}

	// on error, doesnt show helper/usage
	awsCmd.SilenceUsage = true

	// wire up new commands
	awsCmd.AddCommand(Create(), Destroy())

	return awsCmd
}

func Create() *cobra.Command {
	createCmd := &cobra.Command{
		Use:              "create",
		Short:            "create the kubefirst platform running in aws",
		TraverseChildren: true,
		RunE:             createAws,
	}

	// todo review defaults and update descriptions
	createCmd.Flags().StringVar(&alertsEmailFlag, "alerts-email", "", "email address for let's encrypt certificate notifications (required)")
	createCmd.MarkFlagRequired("alerts-email")
	createCmd.Flags().StringVar(&cloudRegionFlag, "cloud-region", "us-east-1", "the aws region to provision infrastructure in")
	createCmd.Flags().StringVar(&clusterNameFlag, "cluster-name", "kubefirst", "the name of the cluster to create")
	createCmd.Flags().StringVar(&clusterTypeFlag, "cluster-type", "mgmt", "the type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().StringVar(&domainNameFlag, "domain-name", "", "the Route53 hosted zone name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com) (required)")
	createCmd.MarkFlagRequired("domain-name")
	createCmd.Flags().BoolVar(&dryRun, "dry-run", false, "don't execute the installation")
	createCmd.Flags().StringVar(&githubOwnerFlag, "github-owner", "", "the github owner of the new gitops and metaphor repositories")
	createCmd.MarkFlagRequired("github-owner")
	createCmd.Flags().StringVar(&gitopsTemplateBranchFlag, "gitops-template-branch", "aws-domain-refactor-5", "the branch to clone for the gitops-template repository")
	createCmd.Flags().StringVar(&gitopsTemplateURLFlag, "gitops-template-url", "https://github.com/kubefirst/gitops-template.git", "the fully qualified url to the gitops-template repository to clone")
	createCmd.Flags().StringVar(&kbotPasswordFlag, "kbot-password", "", "the default password to use for the kbot user")
	createCmd.Flags().StringVar(&metaphorTemplateBranchFlag, "metaphor-template-branch", "main", "the branch to clone for the metaphor-template repository")
	createCmd.Flags().StringVar(&metaphorTemplateURLFlag, "metaphor-template-url", "https://github.com/kubefirst/metaphor-frontend-template.git", "the fully qualified url to the metaphor-template repository to clone")
	createCmd.Flags().BoolVar(&useTelemetryFlag, "use-telemetry", true, "whether to emit telemetry")

	return createCmd
}

func Destroy() *cobra.Command {
	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "destroy the kubefirst platform",
		Long:  "deletes the GitHub resources, aws resources, and local content to re-provision",
		RunE:  destroyAws,
	}

	return destroyCmd
}
