/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"fmt"
	"io"

	"github.com/konstructio/kubefirst-api/pkg/constants"
	"github.com/konstructio/kubefirst/internal/common"
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

type options struct {
	AlertsEmail          string
	CI                   bool
	CloudRegion          string
	ClusterName          string
	ClusterType          string
	DNSProvider          string
	GitHubOrg            string
	GitLabGroup          string
	GitProvider          string
	GitProtocol          string
	GitopsTemplateURL    string
	GitopsTemplateBranch string
	DomainName           string
	SubdomainName        string
	UseTelemetry         bool
	ECR                  bool
	NodeType             string
	NodeCount            string
	InstallCatalogApps   string
	InstallKubefirstPro  bool
}

func NewCommand(logger common.Logger, writer io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aws",
		Short: "kubefirst aws installation",
		Long:  "kubefirst aws",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("To learn more about aws in kubefirst, run:")
			fmt.Println("  kubefirst aws help")
		},
	}

	service := Service{
		logger,
		writer,
	}

	// wire up new commands
	cmd.AddCommand(Create(service), Destroy(), Quota(), RootCredentials())

	return cmd
}

func Create(service Service) *cobra.Command {
	var opts options

	createCmd := &cobra.Command{
		Use:              "create",
		Short:            "create the kubefirst platform running in aws",
		TraverseChildren: true,
		RunE:             service.createAws,
	}

	awsDefaults := constants.GetCloudDefaults().Aws

	// todo review defaults and update descriptions
	createCmd.Flags().StringVar(&opts.AlertsEmail, "alerts-email", "", "email address for let's encrypt certificate notifications (required)")
	createCmd.MarkFlagRequired("alerts-email")
	createCmd.Flags().BoolVar(&ciFlag, "ci", false, "if running kubefirst in ci, set this flag to disable interactive features")
	createCmd.Flags().StringVar(&opts.CloudRegion, "cloud-region", "us-east-1", "the aws region to provision infrastructure in")
	createCmd.Flags().StringVar(&opts.ClusterName, "cluster-name", "kubefirst", "the name of the cluster to create")
	createCmd.Flags().StringVar(&opts.ClusterType, "cluster-type", "mgmt", "the type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().StringVar(&opts.NodeCount, "node-count", awsDefaults.NodeCount, "the node count for the cluster")
	createCmd.Flags().StringVar(&opts.NodeType, "node-type", awsDefaults.InstanceSize, "the instance size of the cluster to create")
	createCmd.Flags().StringVar(&opts.DNSProvider, "dns-provider", "aws", fmt.Sprintf("the dns provider - one of: %q", supportedDNSProviders))
	createCmd.Flags().StringVar(&opts.SubdomainName, "subdomain", "", "the subdomain to use for DNS records (Cloudflare)")
	createCmd.Flags().StringVar(&opts.DomainName, "domain-name", "", "the Route53/Cloudflare hosted zone name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com) (required)")
	createCmd.MarkFlagRequired("domain-name")
	createCmd.Flags().StringVar(&opts.GitProvider, "git-provider", "github", fmt.Sprintf("the git provider - one of: %q", supportedGitProviders))
	createCmd.Flags().StringVar(&opts.GitProtocol, "git-protocol", "ssh", fmt.Sprintf("the git protocol - one of: %q", supportedGitProtocolOverride))
	createCmd.Flags().StringVar(&opts.GitHubOrg, "github-org", "", "the GitHub organization for the new gitops and metaphor repositories - required if using github")
	createCmd.Flags().StringVar(&opts.GitLabGroup, "gitlab-group", "", "the GitLab group for the new gitops and metaphor projects - required if using gitlab")
	createCmd.Flags().StringVar(&opts.GitopsTemplateBranch, "gitops-template-branch", "", "the branch to clone for the gitops-template repository")
	createCmd.Flags().StringVar(&opts.GitopsTemplateURL, "gitops-template-url", "https://github.com/konstructio/gitops-template.git", "the fully qualified url to the gitops-template repository to clone")
	createCmd.Flags().StringVar(&opts.InstallCatalogApps, "install-catalog-apps", "", "comma separated values to install after provision")
	createCmd.Flags().BoolVar(&opts.UseTelemetry, "use-telemetry", true, "whether to emit telemetry")
	createCmd.Flags().BoolVar(&opts.ECR, "ecr", false, "whether or not to use ecr vs the git provider")
	createCmd.Flags().BoolVar(&opts.InstallKubefirstPro, "install-kubefirst-pro", true, "whether or not to install kubefirst pro")

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
