/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package akamai

import (
	"fmt"

	"github.com/konstructio/kubefirst-api/pkg/constants"
	"github.com/konstructio/kubefirst/internal/common"
	"github.com/spf13/cobra"
)

var (
	// Supported providers
	supportedDNSProviders = []string{"cloudflare"}
	supportedGitProviders = []string{"github", "gitlab"}
	// Supported git protocols
	supportedGitProtocolOverride = []string{"https", "ssh"}
)

func NewCommand() *cobra.Command {
	akamaiCmd := &cobra.Command{
		Use:   "akamai",
		Short: "kubefirst akamai installation",
		Long:  "kubefirst akamai",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("To learn more about akamai in kubefirst, run:")
			fmt.Println("  kubefirst beta akamai --help")

			return nil
		},
	}

	// wire up new commands
	akamaiCmd.AddCommand(Create(), Destroy(), RootCredentials())

	return akamaiCmd
}

func Create() *cobra.Command {
	createCmd := &cobra.Command{
		Use:              "create",
		Short:            "create the kubefirst platform running on akamai kubernetes",
		TraverseChildren: true,
		RunE:             createAkamai,
	}

	akamaiDefaults := constants.GetCloudDefaults().Akamai

	// todo review defaults and update descriptions
	createCmd.Flags().String("alerts-email", "", "email address for let's encrypt certificate notifications (required)")
	createCmd.MarkFlagRequired("alerts-email")
	createCmd.Flags().Bool("ci", false, "if running kubefirst in ci, set this flag to disable interactive features")
	createCmd.Flags().String("cloud-region", "us-central", "the akamai region to provision infrastructure in")
	createCmd.Flags().String("cluster-name", "kubefirst", "the name of the cluster to create")
	createCmd.Flags().String("cluster-type", "mgmt", "the type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().String("node-count", akamaiDefaults.NodeCount, "the node count for the cluster")
	createCmd.Flags().String("node-type", akamaiDefaults.InstanceSize, "the instance size of the cluster to create")
	createCmd.Flags().String("dns-provider", "cloudflare", fmt.Sprintf("the dns provider - one of: %q", supportedDNSProviders))
	createCmd.Flags().String("subdomain", "", "the subdomain to use for DNS records (Cloudflare)")
	createCmd.Flags().String("domain-name", "", "the DNS Name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com) (required)")
	createCmd.MarkFlagRequired("domain-name")
	createCmd.Flags().String("git-provider", "github", fmt.Sprintf("the git provider - one of: %q", supportedGitProviders))
	createCmd.Flags().String("git-protocol", "https", fmt.Sprintf("the git protocol - one of: %q", supportedGitProtocolOverride))
	createCmd.Flags().String("github-org", "", "the GitHub organization for the new gitops and metaphor repositories - required if using github")
	createCmd.Flags().String("gitlab-group", "", "the GitLab group for the new gitops and metaphor projects - required if using gitlab")
	createCmd.Flags().String("gitops-template-branch", "", "the branch to clone for the gitops-template repository")
	createCmd.Flags().String("gitops-template-url", "https://github.com/konstructio/gitops-template.git", "the fully qualified url to the gitops-template repository to clone")
	createCmd.Flags().String("install-catalog-apps", "", "comma separated values to install after provision")
	createCmd.Flags().Bool("use-telemetry", true, "whether to emit telemetry")
	createCmd.Flags().Bool("install-kubefirst-pro", true, "whether or not to install kubefirst pro")

	return createCmd
}

func Destroy() *cobra.Command {
	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "destroy the kubefirst platform",
		Long:  "destroy the kubefirst platform running in akamai and remove all resources",
		RunE:  common.Destroy,
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

	authCmd.Flags().Bool("argocd", false, "copy the ArgoCD password to the clipboard (optional)")
	authCmd.Flags().Bool("kbot", false, "copy the kbot password to the clipboard (optional)")
	authCmd.Flags().Bool("vault", false, "copy the vault password to the clipboard (optional)")

	return authCmd
}
