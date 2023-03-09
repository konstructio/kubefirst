package civo

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Create
	alertsEmailFlag          string
	cloudRegionFlag          string
	clusterNameFlag          string
	clusterTypeFlag          string
	dryRun                   bool
	githubOwnerFlag          string
	gitlabOwnerFlag          string
	gitProviderFlag          string
	gitopsTemplateURLFlag    string
	gitopsTemplateBranchFlag string
	domainNameFlag           string
	kbotPasswordFlag         string
	useTelemetryFlag         bool

	// Supported git providers
	supportedGitProviders = []string{"github", "gitlab"}
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
	createCmd.Flags().StringVar(&gitProviderFlag, "git-provider", "github", fmt.Sprintf("the git provider - one of: %s", supportedGitProviders))
	createCmd.Flags().StringVar(&githubOwnerFlag, "github-owner", "", "the GitHub owner of the new gitops and metaphor repositories - required if using github")
	createCmd.Flags().StringVar(&gitlabOwnerFlag, "gitlab-owner", "", "the GitLab owner (group) of the new gitops and metaphor projects - required if using gitlab")
	createCmd.Flags().StringVar(&gitopsTemplateBranchFlag, "gitops-template-branch", "v2-directory-shift", "the branch to clone for the gitops-template repository")
	createCmd.Flags().StringVar(&gitopsTemplateURLFlag, "gitops-template-url", "https://github.com/kubefirst/gitops-template.git", "the fully qualified url to the gitops-template repository to clone")
	createCmd.Flags().StringVar(&kbotPasswordFlag, "kbot-password", "", "the default password to use for the kbot user")
	createCmd.Flags().BoolVar(&useTelemetryFlag, "use-telemetry", true, "whether to emit telemetry")

	return createCmd
}

func Destroy() *cobra.Command {
	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "destroy the kubefirst platform",
		Long:  "destroy the kubefirst platform running in civo and remove all resources",
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

	quotaCmd.Flags().StringVar(&cloudRegionFlag, "cloud-region", "NYC1", "the civo region to monitor quotas in")

	return quotaCmd
}
