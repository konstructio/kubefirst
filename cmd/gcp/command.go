/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gcp

import (
	"fmt"

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
	gcpProjectFlag           string
	githubOrgFlag            string
	gitlabGroupFlag          string
	gitProviderFlag          string
	gitProtocolFlag          string
	gitopsTemplateURLFlag    string
	gitopsTemplateBranchFlag string
	useTelemetryFlag         bool

	// RootCredentials
	copyArgoCDPasswordToClipboardFlag bool
	copyKbotPasswordToClipboardFlag   bool
	copyVaultPasswordToClipboardFlag  bool

	// Supported providers
	supportedDNSProviders = []string{"gcp", "cloudflare"}
	supportedGitProviders = []string{"github", "gitlab"}

	// Supported git providers
	supportedGitProtocolOverride = []string{"https", "ssh"}
)

func NewCommand() *cobra.Command {
	gcpCmd := &cobra.Command{
		Use:   "gcp",
		Short: "kubefirst GCP installation",
		Long:  "kubefirst gcp",
	}

	// on error, doesnt show helper/usage
	gcpCmd.SilenceUsage = true

	// wire up new commands
	gcpCmd.AddCommand(Create(), Destroy(), RootCredentials())

	return gcpCmd
}

func Create() *cobra.Command {
	createCmd := &cobra.Command{
		Use:              "create",
		Short:            "create the kubefirst platform running on GCP kubernetes",
		TraverseChildren: true,
		RunE:             createGCP,
	}

	// todo review defaults and update descriptions
	createCmd.Flags().StringVar(&alertsEmailFlag, "alerts-email", "", "email address for let's encrypt certificate notifications (required)")
	createCmd.MarkFlagRequired("alerts-email")
	createCmd.Flags().BoolVar(&ciFlag, "ci", false, "if running kubefirst in ci, set this flag to disable interactive features")
	createCmd.Flags().StringVar(&cloudRegionFlag, "cloud-region", "us-east1", "the GCP region to provision infrastructure in")
	createCmd.Flags().StringVar(&clusterNameFlag, "cluster-name", "kubefirst", "the name of the cluster to create")
	createCmd.Flags().StringVar(&clusterTypeFlag, "cluster-type", "mgmt", "the type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().StringVar(&dnsProviderFlag, "dns-provider", "gcp", fmt.Sprintf("the dns provider - one of: %s", supportedDNSProviders))
	createCmd.Flags().StringVar(&domainNameFlag, "domain-name", "", "the GCP DNS Name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com) (required)")
	createCmd.MarkFlagRequired("domain-name")
	createCmd.Flags().StringVar(&gcpProjectFlag, "gcp-project", "", "gcp project id (required)")
	createCmd.MarkFlagRequired("gcp-project")
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
		Long:  "destroy the kubefirst platform running in GCP and remove all resources",
		RunE:  destroyGCP,
	}

	return destroyCmd
}

func RootCredentials() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "root-credentials",
		Short: "retrieve root authentication information for platform components",
		Long:  "retrieve root authentication information for platform components",
		RunE:  getGCPRootCredentials,
	}

	authCmd.Flags().BoolVar(&copyArgoCDPasswordToClipboardFlag, "argocd", false, "copy the argocd password to the clipboard (optional)")
	authCmd.Flags().BoolVar(&copyKbotPasswordToClipboardFlag, "kbot", false, "copy the kbot password to the clipboard (optional)")
	authCmd.Flags().BoolVar(&copyVaultPasswordToClipboardFlag, "vault", false, "copy the vault password to the clipboard (optional)")

	return authCmd
}
