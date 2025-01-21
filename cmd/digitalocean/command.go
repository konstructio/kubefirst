/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package digitalocean

import (
	"fmt"

	"github.com/konstructio/kubefirst-api/pkg/constants"
	"github.com/konstructio/kubefirst/internal/common"
	"github.com/spf13/cobra"
)

var (
	// Supported providers
	supportedDNSProviders = []string{"digitalocean", "cloudflare"}
	supportedGitProviders = []string{"github", "gitlab"}
	// Supported git protocols
	supportedGitProtocolOverride = []string{"https", "ssh"}
)

func NewCommand() *cobra.Command {
	digitaloceanCmd := &cobra.Command{
		Use:   "digitalocean",
		Short: "Kubefirst DigitalOcean installation",
		Long:  "Kubefirst DigitalOcean",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("To learn more about DigitalOcean in Kubefirst, run:")
			fmt.Println("  kubefirst digitalocean --help")
		},
	}

	// on error, doesnt show helper/usage
	digitaloceanCmd.SilenceUsage = true

	// wire up new commands
	digitaloceanCmd.AddCommand(Create(), Destroy(), RootCredentials())

	return digitaloceanCmd
}

func Create() *cobra.Command {
	createCmd := &cobra.Command{
		Use:              "create",
		Short:            "create the Kubefirst platform running on DigitalOcean Kubernetes",
		TraverseChildren: true,
		RunE:             createDigitalocean,
		// PreRun:           common.CheckDocker,
	}

	doDefaults := constants.GetCloudDefaults().DigitalOcean

	// todo review defaults and update descriptions
	createCmd.Flags().String("alerts-email", "", "email address for let's encrypt certificate notifications (required)")
	createCmd.MarkFlagRequired("alerts-email")
	createCmd.Flags().Bool("ci", false, "if running Kubefirst in CI, set this flag to disable interactive features")
	createCmd.Flags().String("cloud-region", "nyc3", "the DigitalOcean region to provision infrastructure in")
	createCmd.Flags().String("cluster-name", "kubefirst", "the name of the cluster to create")
	createCmd.Flags().String("cluster-type", "mgmt", "the type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().String("node-count", doDefaults.NodeCount, "the node count for the cluster")
	createCmd.Flags().String("node-type", doDefaults.InstanceSize, "the instance size of the cluster to create")
	createCmd.Flags().String("dns-provider", "digitalocean", fmt.Sprintf("the dns provider - one of: %q", supportedDNSProviders))
	createCmd.Flags().String("subdomain", "", "the subdomain to use for DNS records (Cloudflare)")
	createCmd.Flags().String("domain-name", "", "the DigitalOcean DNS Name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com) (required)")
	createCmd.MarkFlagRequired("domain-name")
	createCmd.Flags().String("git-provider", "github", fmt.Sprintf("the git provider - one of: %q", supportedGitProviders))
	createCmd.Flags().String("git-protocol", "ssh", fmt.Sprintf("the git protocol - one of: %q", supportedGitProtocolOverride))
	createCmd.Flags().String("github-org", "", "the GitHub organization for the new gitops and metaphor repositories - required if using GitHub")
	createCmd.Flags().String("gitlab-group", "", "the GitLab group for the new gitops and metaphor projects - required if using GitLab")
	createCmd.Flags().String("gitops-template-branch", "", "the branch to clone for the gitops-template repository")
	createCmd.Flags().String("gitops-template-url", "https://github.com/konstructio/gitops-template.git", "the fully qualified url to the gitops-template repository to clone")
	createCmd.Flags().String("install-catalog-apps", "", "comma separated values to install after provision")
	createCmd.Flags().Bool("use-telemetry", true, "whether to emit telemetry")
	createCmd.Flags().Bool("install-kubefirst-pro", true, "whether or not to install Kubefirst Pro")

	return createCmd
}

func Destroy() *cobra.Command {
	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "destroy the Kubefirst platform",
		Long:  "destroy the Kubefirst platform running in DigitalOcean and remove all resources",
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

	authCmd.Flags().Bool("argocd", false, "copy the ArgoCD password to the clipboard (optional)")
	authCmd.Flags().Bool("kbot", false, "copy the kbot password to the clipboard (optional)")
	authCmd.Flags().Bool("vault", false, "copy the vault password to the clipboard (optional)")

	return authCmd
}
