/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3s

import (
	"fmt"

	"github.com/konstructio/kubefirst/internal/common"
	"github.com/spf13/cobra"
)

var (
	// Create
	// TODO: add ssh key flag to connect on k3s targets
	alertsEmailFlag          string
	ciFlag                   bool
	cloudRegionFlag          string
	nodeTypeFlag             string
	nodeCountFlag            string
	clusterNameFlag          string
	clusterTypeFlag          string
	k3sServersPrivateIpsFlag []string
	k3sServersPublicIpsFlag  []string
	k3sSshUserflag           string
	k3sSshPrivateKeyflag     string
	K3sServersArgsFlags      []string
	dnsProviderFlag          string
	subdomainNameFlag        string
	domainNameFlag           string
	githubOrgFlag            string
	gitlabGroupFlag          string
	gitProviderFlag          string
	gitProtocolFlag          string
	gitopsTemplateURLFlag    string
	gitopsTemplateBranchFlag string
	gitopsRepoName           string
	metaphorRepoName         string
	adminTeamName            string
	developerTeamName        string
	installCatalogApps       string
	useTelemetryFlag         bool
	forceDestroyFlag         bool
	installKubefirstProFlag  bool

	// RootCredentials
	copyArgoCDPasswordToClipboardFlag bool
	copyKbotPasswordToClipboardFlag   bool
	copyVaultPasswordToClipboardFlag  bool

	// Supported providers
	supportedDNSProviders = []string{"cloudflare"}
	supportedGitProviders = []string{"github", "gitlab"}

	// Supported git providers
	supportedGitProtocolOverride = []string{"https", "ssh"}
)

func NewCommand() *cobra.Command {
	k3sCmd := &cobra.Command{
		Use:   "k3s",
		Short: "kubefirst K3s installation",
		Long:  "kubefirst k3s on premises installation",
	}

	// on error, doesnt show helper/usage
	k3sCmd.SilenceUsage = true

	// wire up new commands
	k3sCmd.AddCommand(Create(), Destroy(), RootCredentials())

	return k3sCmd
}

func Create() *cobra.Command {
	createCmd := &cobra.Command{
		Use:              "create",
		Short:            "create the kubefirst platform running on premise",
		TraverseChildren: true,
		RunE:             createK3s,
		// PreRun:           common.CheckDocker,
	}

	// todo review defaults and update descriptions
	createCmd.Flags().StringVar(&alertsEmailFlag, "alerts-email", "", "email address for let's encrypt certificate notifications (required)")
	createCmd.MarkFlagRequired("alerts-email")
	createCmd.Flags().BoolVar(&ciFlag, "ci", false, "if running kubefirst in ci, set this flag to disable interactive features")
	createCmd.Flags().StringVar(&cloudRegionFlag, "cloud-region", "on-premise", "NOT USED, PRESENT FOR COMPATIBILITY ISSUE")
	createCmd.Flags().StringVar(&nodeTypeFlag, "node-type", "on-premise", "NOT USED, PRESENT FOR COMPATIBILITY ISSUE")
	createCmd.Flags().StringVar(&nodeCountFlag, "node-count", "3", "NOT USED, PRESENT FOR COMPATIBILITY ISSUE")
	createCmd.Flags().StringVar(&clusterNameFlag, "cluster-name", "kubefirst", "the name of the cluster to create")
	createCmd.Flags().StringVar(&clusterTypeFlag, "cluster-type", "mgmt", "the type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().StringSliceVar(&k3sServersPrivateIpsFlag, "servers-private-ips", []string{}, "the list of k3s (servers) private ip x.x.x.x,y.y.y.y comma separated  (required)")
	createCmd.MarkFlagRequired("servers-private-ips")
	createCmd.Flags().StringSliceVar(&k3sServersPublicIpsFlag, "servers-public-ips", []string{}, "the list of k3s (servers) public ip x.x.x.x,y.y.y.y comma separated  (required)")
	createCmd.Flags().StringSliceVar(&K3sServersArgsFlags, "servers-args", []string{"--disable traefik", "--write-kubeconfig-mode 644"}, "list of k3s extras args to add to the k3s server installation,comma separated in between quote, if --servers-publis-ips <VALUES> --tls-san <VALUES> is added to default --servers-args")
	createCmd.Flags().StringVar(&k3sSshUserflag, "ssh-user", "root", "the user used to log into servers with ssh connection")
	createCmd.Flags().StringVar(&k3sSshPrivateKeyflag, "ssh-privatekey", "", "the private key used to log into servers with ssh connection")
	createCmd.MarkFlagRequired("ssh-privatekey")
	createCmd.Flags().StringVar(&dnsProviderFlag, "dns-provider", "cloudflare", fmt.Sprintf("the dns provider - one of: %s", supportedDNSProviders))
	createCmd.Flags().StringVar(&subdomainNameFlag, "subdomain", "", "the subdomain to use for DNS records (Cloudflare)")
	createCmd.Flags().StringVar(&domainNameFlag, "domain-name", "", "the cloudProvider DNS Name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com) (required)")
	createCmd.Flags().StringVar(&gitProviderFlag, "git-provider", "github", fmt.Sprintf("the git provider - one of: %s", supportedGitProviders))
	createCmd.Flags().StringVar(&gitProtocolFlag, "git-protocol", "ssh", fmt.Sprintf("the git protocol - one of: %s", supportedGitProtocolOverride))
	createCmd.Flags().StringVar(&githubOrgFlag, "github-org", "", "the GitHub organization for the new gitops and metaphor repositories - required if using github")
	createCmd.Flags().StringVar(&gitlabGroupFlag, "gitlab-group", "", "the GitLab group for the new gitops and metaphor projects - required if using gitlab")
	createCmd.Flags().StringVar(&gitopsTemplateBranchFlag, "gitops-template-branch", "", "the branch to clone for the gitops-template repository")
	createCmd.Flags().StringVar(&gitopsTemplateURLFlag, "gitops-template-url", "https://github.com/kubefirst/gitops-template.git", "the fully qualified url to the gitops-template repository to clone")
	createCmd.Flags().StringVar(&installCatalogApps, "install-catalog-apps", "", "comma seperated values to install after provision")
	createCmd.Flags().BoolVar(&useTelemetryFlag, "use-telemetry", true, "whether to emit telemetry")
	createCmd.Flags().BoolVar(&forceDestroyFlag, "force-destroy", false, "allows force destruction on objects (helpful for test environments, defaults to false)")
	createCmd.Flags().BoolVar(&installKubefirstProFlag, "install-kubefirst-pro", true, "whether or not to install kubefirst pro")
	createCmd.Flags().StringVar(&gitopsRepoName, "gitops-repo-name", "gitops", "the custom gitops name")
	createCmd.Flags().StringVar(&metaphorRepoName, "metaphor-repo-name", "metaphor", "the custom metaphor name")
	createCmd.Flags().StringVar(&adminTeamName, "admin-team-name", "admins", "admin team name for this repo")
	createCmd.Flags().StringVar(&developerTeamName, "developer-team-name", "developers", "developer team name for this repo")

	return createCmd
}

func Destroy() *cobra.Command {
	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "destroy the kubefirst platform",
		Long:  "destroy the kubefirst platform running in k3s cluster",
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

	authCmd.Flags().BoolVar(&copyArgoCDPasswordToClipboardFlag, "argocd", false, "copy the argocd password to the clipboard (optional)")
	authCmd.Flags().BoolVar(&copyKbotPasswordToClipboardFlag, "kbot", false, "copy the kbot password to the clipboard (optional)")
	authCmd.Flags().BoolVar(&copyVaultPasswordToClipboardFlag, "vault", false, "copy the vault password to the clipboard (optional)")

	return authCmd
}
