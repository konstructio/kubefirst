/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

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
	githubOrgFlag            string
	gitlabGroupFlag          string
	gitProviderFlag          string
	gitProtocolFlag          string
	gitopsTemplateURLFlag    string
	gitopsTemplateBranchFlag string
	domainNameFlag           string
	subdomainNameFlag        string
	useTelemetryFlag         bool
	ecrFlag                  bool
	nodeTypeFlag             string
	nodeCountFlag            string
	installCatalogApps       string
	installKubefirstProFlag  bool

	// Supported argument arrays
	supportedDNSProviders        = []string{"aws", "cloudflare"}
	supportedGitProviders        = []string{"github", "gitlab"}
	supportedGitProtocolOverride = []string{"https", "ssh"}
)

func NewCommand() *cobra.Command {
	awsCmd := &cobra.Command{
		Use:   "aws",
		Short: "kubefirst aws installation",
		Long:  "kubefirst aws",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("To learn more about aws in kubefirst, run:")
			fmt.Println("  kubefirst help")

			if progress.Progress != nil {
				progress.Progress.Quit()
			}
		},
	}

	// wire up new commands
	awsCmd.AddCommand(Create(), Destroy(), Quota(), RootCredentials())

	return awsCmd
}

func Create() *cobra.Command {
	createCmd := &cobra.Command{
		Use:              "create",
		Short:            "create the kubefirst platform running in aws",
		TraverseChildren: true,
		RunE:             createAws,
		// PreRun:           common.CheckDocker,
	}

	awsDefaults := constants.GetCloudDefaults().Aws

	// todo review defaults and update descriptions
	createCmd.Flags().StringVar(&alertsEmailFlag, "alerts-email", "", "email address for let's encrypt certificate notifications (required)")
	createCmd.MarkFlagRequired("alerts-email")

	createCmd.Flags().StringVar(&cloudRegionFlag, "cloud-region", "us-east-1", "the aws region to provision infrastructure in")
	createCmd.Flags().StringVar(&clusterNameFlag, "cluster-name", "kubefirst", "the name of the cluster to create")
	createCmd.Flags().StringVar(&clusterTypeFlag, "cluster-type", "mgmt", "the type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().StringVar(&nodeCountFlag, "node-count", awsDefaults.NodeCount, "the node count for the cluster")
	createCmd.Flags().StringVar(&nodeTypeFlag, "node-type", awsDefaults.InstanceSize, "the instance size of the cluster to create")
	createCmd.Flags().StringVar(&dnsProviderFlag, "dns-provider", "aws", fmt.Sprintf("the dns provider - one of: %q", supportedDNSProviders))
	createCmd.Flags().StringVar(&subdomainNameFlag, "subdomain", "", "the subdomain to use for DNS records (Cloudflare)")
	createCmd.Flags().StringVar(&domainNameFlag, "domain-name", "", "the Route53/Cloudflare hosted zone name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com) (required)")
	createCmd.MarkFlagRequired("domain-name")
	createCmd.Flags().StringVar(&gitProviderFlag, "git-provider", "github", fmt.Sprintf("the git provider - one of: %q", supportedGitProviders))
	createCmd.Flags().StringVar(&gitProtocolFlag, "git-protocol", "ssh", fmt.Sprintf("the git protocol - one of: %q", supportedGitProtocolOverride))
	createCmd.Flags().StringVar(&githubOrgFlag, "github-org", "", "the GitHub organization for the new gitops and metaphor repositories - required if using github")
	createCmd.Flags().StringVar(&gitlabGroupFlag, "gitlab-group", "", "the GitLab group for the new gitops and metaphor projects - required if using gitlab")
	createCmd.Flags().StringVar(&gitopsTemplateBranchFlag, "gitops-template-branch", "", "the branch to clone for the gitops-template repository")
	createCmd.Flags().StringVar(&gitopsTemplateURLFlag, "gitops-template-url", "https://github.com/konstructio/gitops-template.git", "the fully qualified url to the gitops-template repository to clone")
	createCmd.Flags().StringVar(&installCatalogApps, "install-catalog-apps", "", "comma separated values to install after provision")
	createCmd.Flags().BoolVar(&useTelemetryFlag, "use-telemetry", true, "whether to emit telemetry")
	createCmd.Flags().BoolVar(&ecrFlag, "ecr", false, "whether or not to use ecr vs the git provider")
	createCmd.Flags().BoolVar(&installKubefirstProFlag, "install-kubefirst-pro", true, "whether or not to install kubefirst pro")

	return createCmd
}

func Destroy() *cobra.Command {
	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "destroy the kubefirst platform",
		Long:  "deletes the GitHub resources, aws resources, and local content to re-provision",
		RunE:  common.Destroy,
		// PreRun: common.CheckDocker,
	}

	return destroyCmd
}

func Quota() *cobra.Command {
	quotaCmd := &cobra.Command{
		Use:   "quota",
		Short: "Check aws quota status",
		Long:  "Check aws quota status. By default, only ones close to limits will be shown.",
		RunE:  evalAwsQuota,
	}

	quotaCmd.Flags().StringVar(&cloudRegionFlag, "cloud-region", "us-east-1", "the aws region to provision infrastructure in")

	return quotaCmd
}

func RootCredentials() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "root-credentials",
		Short: "retrieve root authentication information for platform components",
		Long:  "retrieve root authentication information for platform components",
		RunE:  common.GetRootCredentials,
	}

	return authCmd
}
