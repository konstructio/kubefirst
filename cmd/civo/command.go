/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"fmt"

	"github.com/konstructio/kubefirst-api/pkg/constants"
	"github.com/konstructio/kubefirst/internal/common"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/spf13/cobra"
)

var (
	// Create
	alertsEmailFlag          string
	cloudRegionFlag          string
	clusterNameFlag          string
	clusterTypeFlag          string
	dnsProviderFlag          string
	subdomainNameFlag        string
	domainNameFlag           string
	githubOrgFlag            string
	gitlabGroupFlag          string
	gitProviderFlag          string
	gitProtocolFlag          string
	gitopsTemplateURLFlag    string
	gitopsTemplateBranchFlag string
	useTelemetryFlag         bool
	nodeTypeFlag             string
	nodeCountFlag            string
	installCatalogApps       string
	installKubefirstProFlag  bool

	// RootCredentials
	copyArgoCDPasswordToClipboardFlag bool
	copyKbotPasswordToClipboardFlag   bool
	copyVaultPasswordToClipboardFlag  bool

	// Supported providers
	supportedDNSProviders = []string{"civo", "cloudflare"}
	supportedGitProviders = []string{"github", "gitlab"}
	// Supported git protocols
	supportedGitProtocolOverride = []string{"https", "ssh"}
)

func NewCommand() *cobra.Command {
	civoCmd := &cobra.Command{
		Use:   "civo",
		Short: "Kubefirst Civo installation",
		Long:  "Kubefirst Civo",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("To learn more about Civo in Kubefirst, run:")
			fmt.Println("  kubefirst civo --help")

			if progress.Progress != nil {
				progress.Progress.Quit()
			}
		},
	}

	// wire up new commands
	civoCmd.AddCommand(BackupSSL(), Create(), Destroy(), Quota(), RootCredentials())

	return civoCmd
}

func BackupSSL() *cobra.Command {
	backupSSLCmd := &cobra.Command{
		Use:   "backup-ssl",
		Short: "Backup the cluster resources related to TLS certificates from cert-manager",
		Long:  "Kubefirst uses a combination of external-dns, ingress-nginx, and cert-manager for provisioning automated TLS certificates for services with an ingress. This command will backup all the Kubernetes resources to restore in a new cluster with the same domain name",
		RunE:  backupCivoSSL,
	}

	return backupSSLCmd
}

func Create() *cobra.Command {
	createCmd := &cobra.Command{
		Use:              "create",
		Short:            "Create the Kubefirst platform running on Civo Kubernetes",
		TraverseChildren: true,
		RunE:             createCivo,
		// PreRun:           common.CheckDocker,
	}

	civoDefaults := constants.GetCloudDefaults().Civo

	// todo review defaults and update descriptions
	createCmd.Flags().StringVar(&alertsEmailFlag, "alerts-email", "", "Email address for Let's Encrypt certificate notifications (required)")
	createCmd.MarkFlagRequired("alerts-email")
	createCmd.Flags().StringVar(&cloudRegionFlag, "cloud-region", "NYC1", "The Civo region to provision infrastructure in")
	createCmd.Flags().StringVar(&clusterNameFlag, "cluster-name", "kubefirst", "The name of the cluster to create")
	createCmd.Flags().StringVar(&clusterTypeFlag, "cluster-type", "mgmt", "The type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().StringVar(&nodeCountFlag, "node-count", civoDefaults.NodeCount, "The node count for the cluster")
	createCmd.Flags().StringVar(&nodeTypeFlag, "node-type", civoDefaults.InstanceSize, "The instance size of the cluster to create")
	createCmd.Flags().StringVar(&dnsProviderFlag, "dns-provider", "civo", fmt.Sprintf("The DNS provider - one of: %s", supportedDNSProviders))
	createCmd.Flags().StringVar(&subdomainNameFlag, "subdomain", "", "The subdomain to use for DNS records (Cloudflare)")
	createCmd.Flags().StringVar(&domainNameFlag, "domain-name", "", "The Civo DNS Name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com) (required)")
	createCmd.MarkFlagRequired("domain-name")
	createCmd.Flags().StringVar(&gitProviderFlag, "git-provider", "github", fmt.Sprintf("The git provider - one of: %s", supportedGitProviders))
	createCmd.Flags().StringVar(&gitProtocolFlag, "git-protocol", "ssh", fmt.Sprintf("The git protocol - one of: %s", supportedGitProtocolOverride))
	createCmd.Flags().StringVar(&githubOrgFlag, "github-org", "", "The GitHub organization for the new GitOps and Metaphor repositories - required if using GitHub")
	createCmd.Flags().StringVar(&gitlabGroupFlag, "gitlab-group", "", "The GitLab group for the new GitOps and Metaphor projects - required if using GitLab")
	createCmd.Flags().StringVar(&gitopsTemplateBranchFlag, "gitops-template-branch", "", "The branch to clone for the gitops-template repository")
	createCmd.Flags().StringVar(&gitopsTemplateURLFlag, "gitops-template-url", "https://github.com/konstructio/gitops-template.git", "The fully qualified URL to the gitops-template repository to clone")
	createCmd.Flags().StringVar(&installCatalogApps, "install-catalog-apps", "", "Comma separated values to install after provision")
	createCmd.Flags().BoolVar(&useTelemetryFlag, "use-telemetry", true, "Whether to emit telemetry")
	createCmd.Flags().BoolVar(&installKubefirstProFlag, "install-kubefirst-pro", true, "Whether or not to install Kubefirst Pro")

	return createCmd
}

func Destroy() *cobra.Command {
	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "Destroy the Kubefirst platform",
		Long:  "Destroy the Kubefirst platform running in Civo and remove all resources",
		RunE:  common.Destroy,
		// PreRun: common.CheckDocker,
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

	quotaCmd.Flags().StringVar(&cloudRegionFlag, "cloud-region", "NYC1", "The Civo region to monitor quotas in")

	return quotaCmd
}

func RootCredentials() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "root-credentials",
		Short: "Retrieve root authentication information for platform components",
		Long:  "Retrieve root authentication information for platform components",
		RunE:  common.GetRootCredentials,
	}

	authCmd.Flags().BoolVar(&copyArgoCDPasswordToClipboardFlag, "argocd", false, "Copy the ArgoCD password to the clipboard (optional)")
	authCmd.Flags().BoolVar(&copyKbotPasswordToClipboardFlag, "kbot", false, "Copy the Kbot password to the clipboard (optional)")
	authCmd.Flags().BoolVar(&copyVaultPasswordToClipboardFlag, "vault", false, "Copy the Vault password to the clipboard (optional)")

	return authCmd
}
