/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"fmt"

	"github.com/kubefirst/kubefirst-api/pkg/constants"
	"github.com/kubefirst/kubefirst/internal/common"
	"github.com/kubefirst/kubefirst/internal/progress"
	"github.com/spf13/cobra"
)

var (
	// Create
	alertsEmailFlag          string
	ciFlag                   bool
	cloudRegionFlag          string
	clusterNameFlag          string
	clusterTypeFlag          string
	dnsProviderFlag          string
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
		Short: "kubefirst civo installation",
		Long:  "kubefirst civo",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("To learn more about civo in kubefirst, run:")
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
		// PreRun:           common.CheckDocker,
	}

	civoDefaults := constants.GetCloudDefaults().Civo

	// todo review defaults and update descriptions
	createCmd.Flags().StringVar(&alertsEmailFlag, "alerts-email", "", "email address for let's encrypt certificate notifications (required)")
	createCmd.MarkFlagRequired("alerts-email")
	createCmd.Flags().BoolVar(&ciFlag, "ci", false, "if running kubefirst in ci, set this flag to disable interactive features")
	createCmd.Flags().StringVar(&cloudRegionFlag, "cloud-region", "NYC1", "the civo region to provision infrastructure in")
	createCmd.Flags().StringVar(&clusterNameFlag, "cluster-name", "kubefirst", "the name of the cluster to create")
	createCmd.Flags().StringVar(&clusterTypeFlag, "cluster-type", "mgmt", "the type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().StringVar(&nodeCountFlag, "node-count", civoDefaults.NodeCount, "the node count for the cluster")
	createCmd.Flags().StringVar(&nodeTypeFlag, "node-type", civoDefaults.InstanceSize, "the instance size of the cluster to create")
	createCmd.Flags().StringVar(&dnsProviderFlag, "dns-provider", "civo", fmt.Sprintf("the dns provider - one of: %s", supportedDNSProviders))
	createCmd.Flags().StringVar(&domainNameFlag, "domain-name", "", "the Civo DNS Name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com) (required)")
	createCmd.MarkFlagRequired("domain-name")
	createCmd.Flags().StringVar(&gitProviderFlag, "git-provider", "github", fmt.Sprintf("the git provider - one of: %s", supportedGitProviders))
	createCmd.Flags().StringVar(&gitProtocolFlag, "git-protocol", "ssh", fmt.Sprintf("the git protocol - one of: %s", supportedGitProtocolOverride))
	createCmd.Flags().StringVar(&githubOrgFlag, "github-org", "", "the GitHub organization for the new gitops and metaphor repositories - required if using github")
	createCmd.Flags().StringVar(&gitlabGroupFlag, "gitlab-group", "", "the GitLab group for the new gitops and metaphor projects - required if using gitlab")
	createCmd.Flags().StringVar(&gitopsTemplateBranchFlag, "gitops-template-branch", "", "the branch to clone for the gitops-template repository")
	createCmd.Flags().StringVar(&gitopsTemplateURLFlag, "gitops-template-url", "https://github.com/kubefirst/gitops-template.git", "the fully qualified url to the gitops-template repository to clone")
	createCmd.Flags().BoolVar(&useTelemetryFlag, "use-telemetry", true, "whether to emit telemetry")

	return createCmd
}

func Destroy() *cobra.Command {
	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "destroy the kubefirst platform",
		Long:  "destroy the kubefirst platform running in civo and remove all resources",
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

	quotaCmd.Flags().StringVar(&cloudRegionFlag, "cloud-region", "NYC1", "the civo region to monitor quotas in")

	return quotaCmd
}

func RootCredentials() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "root-credentials",
		Short: "retrieve root authentication information for platform components",
		Long:  "retrieve root authentication information for platform components",
		RunE:  common.GetRootCredentials,
	}

	authCmd.Flags().BoolVar(&copyArgoCDPasswordToClipboardFlag, "argocd", false, "copy the argocd password to the clipboard (optional)")
	authCmd.Flags().BoolVar(&copyKbotPasswordToClipboardFlag, "kbot", false, "copy the kbot password to the clipboard (optional)")
	authCmd.Flags().BoolVar(&copyVaultPasswordToClipboardFlag, "vault", false, "copy the vault password to the clipboard (optional)")

	return authCmd
}
