package civo

import (
	"github.com/spf13/cobra"
)

var (
	// Create
	alertsEmailFlag            string
	cloudRegionFlag            string
	clusterNameFlag            string
	clusterTypeFlag            string
	dryRun                     bool
	githubOwnerFlag            string
	gitopsTemplateURLFlag      string
	gitopsTemplateBranchFlag   string
	metaphorTemplateBranchFlag string
	metaphorTemplateURLFlag    string
	domainNameFlag             string
	kbotPasswordFlag           string
	useTelemetryFlag           bool

	// Quota
	quotaShowAllFlag bool
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
	civoCmd.AddCommand(BackupSSL(), Create(), Destroy(), Quota())

	return civoCmd
}

func BackupSSL() *cobra.Command {
	backupSSLCmd := &cobra.Command{
		Use:   "backup-ssl",
		Short: "backup the cluster resources related tls certificates from cert-manager",
		Long:  "kubefirst uses a combination of external-dns, ingress-nginx, and cert-manager for provisioning automated tls certificates for services with an ingress. this command will backup all the kubernetes resources to restore in a new cluster with the same domain name",
		RunE:  backupCivoSSL,
	}

	return backupSSLCmd
}

func Create() *cobra.Command {
	createCmd := &cobra.Command{
		Use:              "create",
		Short:            "create the kubefirst platform running on civo kubernetes",
		TraverseChildren: true,
		RunE:             createCivo,
	}

	// todo review defaults and update descriptions
	createCmd.Flags().StringVar(&alertsEmailFlag, "alerts-email", "", "email address for let's encrypt certificate notifications (required)")
	createCmd.MarkFlagRequired("alerts-email")
	createCmd.Flags().StringVar(&cloudRegionFlag, "cloud-region", "NYC1", "the civo region to provision infrastructure in")
	createCmd.Flags().StringVar(&clusterNameFlag, "cluster-name", "kubefirst", "the name of the cluster to create")
	createCmd.Flags().StringVar(&clusterTypeFlag, "cluster-type", "mgmt", "the type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().StringVar(&domainNameFlag, "domain-name", "", "the Civo DNS Name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com) (required)")
	createCmd.MarkFlagRequired("domain-name")
	createCmd.Flags().BoolVar(&dryRun, "dry-run", false, "don't execute the installation")
	createCmd.Flags().StringVar(&githubOwnerFlag, "github-owner", "", "the GitHub owner of the new gitops and metaphor repositories (required)")
	createCmd.MarkFlagRequired("github-owner")
	createCmd.Flags().StringVar(&gitopsTemplateBranchFlag, "gitops-template-branch", "main", "the branch to clone for the gitops-template repository")
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
		Long:  "todo",
		RunE:  destroyCivo,
	}

	return destroyCmd
}

func Quota() *cobra.Command {
	quotaCmd := &cobra.Command{
		Use:   "quota",
		Short: "Check Civo quota status",
		Long:  "Check Civo quota status. By default, only ones close to limits will be shown.",
		RunE:  evalCivoQuota,
	}

	quotaCmd.Flags().BoolVar(&quotaShowAllFlag, "show-all", false, "show all quotas regardless of usage")
	return quotaCmd
}
