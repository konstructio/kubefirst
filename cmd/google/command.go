/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package google

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
	ciFlag                   bool
	cloudRegionFlag          string
	clusterNameFlag          string
	clusterTypeFlag          string
	dnsProviderFlag          string
	domainNameFlag           string
	subdomainNameFlag        string
	googleProjectFlag        string
	githubOrgFlag            string
	gitlabGroupFlag          string
	gitProviderFlag          string
	gitProtocolFlag          string
	gitopsTemplateURLFlag    string
	gitopsTemplateBranchFlag string
	useTelemetryFlag         bool
	forceDestroyFlag         bool
	nodeTypeFlag             string
	nodeCountFlag            string
	installCatalogApps       string
	installKubefirstProFlag  bool

	// RootCredentials
	copyArgoCDPasswordToClipboardFlag bool
	copyKbotPasswordToClipboardFlag   bool
	copyVaultPasswordToClipboardFlag  bool

	// Supported providers
	supportedDNSProviders = []string{"google", "cloudflare"}
	supportedGitProviders = []string{"github", "gitlab"}

	// Supported git providers
	supportedGitProtocolOverride = []string{"https", "ssh"}
)

func NewCommand() *cobra.Command {
	googleCmd := &cobra.Command{
		Use:   "google",
		Short: "kubefirst Google installation",
		Long:  "kubefirst google",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Println("To learn more about google in kubefirst, run:")
			fmt.Println("  kubefirst beta google --help")

			if progress.Progress != nil {
				progress.Progress.Quit()
			}
		},
	}

	// on error, doesnt show helper/usage
	googleCmd.SilenceUsage = true

	// wire up new commands
	googleCmd.AddCommand(Create(), Destroy(), RootCredentials())

	return googleCmd
}

func Create() *cobra.Command {
	createCmd := &cobra.Command{
		Use:              "create",
		Short:            "create the kubefirst platform running on GCP Kubernetes",
		TraverseChildren: true,
		RunE:             createGoogle,
		// PreRun:           common.CheckDocker,
	}

	googleDefaults := constants.GetCloudDefaults().Google

	// todo review defaults and update descriptions
	createCmd.Flags().StringVar(&alertsEmailFlag, "alerts-email", "", "email address for let's encrypt certificate notifications (required)")
	createCmd.MarkFlagRequired("alerts-email")
	createCmd.Flags().BoolVar(&ciFlag, "ci", false, "if running kubefirst in ci, set this flag to disable interactive features")
	createCmd.Flags().StringVar(&cloudRegionFlag, "cloud-region", "us-east1", "the GCP region to provision infrastructure in")
	createCmd.Flags().StringVar(&clusterNameFlag, "cluster-name", "kubefirst", "the name of the cluster to create")
	createCmd.Flags().StringVar(&clusterTypeFlag, "cluster-type", "mgmt", "the type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().StringVar(&nodeCountFlag, "node-count", googleDefaults.NodeCount, "the node count for the cluster")
	createCmd.Flags().StringVar(&nodeTypeFlag, "node-type", googleDefaults.InstanceSize, "the instance size of the cluster to create")
	createCmd.Flags().StringVar(&dnsProviderFlag, "dns-provider", "google", fmt.Sprintf("the dns provider - one of: %q", supportedDNSProviders))
	createCmd.Flags().StringVar(&subdomainNameFlag, "subdomain", "", "the subdomain to use for DNS records (Cloudflare)")
	createCmd.Flags().StringVar(&domainNameFlag, "domain-name", "", "the GCP DNS Name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com) (required)")
	createCmd.MarkFlagRequired("domain-name")
	createCmd.Flags().StringVar(&googleProjectFlag, "google-project", "", "google project id (required)")
	createCmd.MarkFlagRequired("google-project")
	createCmd.Flags().StringVar(&gitProviderFlag, "git-provider", "github", fmt.Sprintf("the git provider - one of: %q", supportedGitProviders))
	createCmd.Flags().StringVar(&gitProtocolFlag, "git-protocol", "ssh", fmt.Sprintf("the git protocol - one of: %q", supportedGitProtocolOverride))
	createCmd.Flags().StringVar(&githubOrgFlag, "github-org", "", "the GitHub organization for the new gitops and metaphor repositories - required if using github")
	createCmd.Flags().StringVar(&gitlabGroupFlag, "gitlab-group", "", "the GitLab group for the new gitops and metaphor projects - required if using gitlab")
	createCmd.Flags().StringVar(&gitopsTemplateBranchFlag, "gitops-template-branch", "", "the branch to clone for the gitops-template repository")
	createCmd.Flags().StringVar(&gitopsTemplateURLFlag, "gitops-template-url", "https://github.com/konstructio/gitops-template.git", "the fully qualified url to the gitops-template repository to clone")
	createCmd.Flags().StringVar(&installCatalogApps, "install-catalog-apps", "", "comma separated values to install after provision")
	createCmd.Flags().BoolVar(&useTelemetryFlag, "use-telemetry", true, "whether to emit telemetry")
	createCmd.Flags().BoolVar(&forceDestroyFlag, "force-destroy", false, "allows force destruction on objects (helpful for test environments, defaults to false)")
	createCmd.Flags().BoolVar(&installKubefirstProFlag, "install-kubefirst-pro", true, "whether or not to install kubefirst pro")

	return createCmd
}

func Destroy() *cobra.Command {
	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "destroy the kubefirst platform",
		Long:  "destroy the kubefirst platform running in Google and remove all resources",
		RunE:  common.Destroy,
		// PreRun: common.CheckDocker,
	}

	return destroyCmd
}

func RootCredentials() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "root-credentials",
		Short: "retrieve root authentication information for platform components",
		Long:  "retrieve root authentication information for platform components",
		RunE:  common.GetRootCredentials,
	}

	authCmd.Flags().BoolVar(&copyArgoCDPasswordToClipboardFlag, "argocd", false, "copy the ArgoCD password to the clipboard (optional)")
	authCmd.Flags().BoolVar(&copyKbotPasswordToClipboardFlag, "kbot", false, "copy the kbot password to the clipboard (optional)")
	authCmd.Flags().BoolVar(&copyVaultPasswordToClipboardFlag, "vault", false, "copy the vault password to the clipboard (optional)")

	return authCmd
}
