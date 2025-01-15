/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vultr

import (
	"fmt"

	"github.com/konstructio/kubefirst-api/pkg/constants"
	"github.com/konstructio/kubefirst/internal/common"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/spf13/cobra"
)

var (
	// Supported providers
	supportedDNSProviders = []string{"vultr", "cloudflare"}
	supportedGitProviders = []string{"github", "gitlab"}
	// Supported git protocols
	supportedGitProtocolOverride = []string{"https", "ssh"}
)

func NewCommand() *cobra.Command {
	vultrCmd := &cobra.Command{
		Use:   "vultr",
		Short: "Kubefirst Vultr installation",
		Long:  "kubefirst vultr",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("To learn more about Vultr in Kubefirst, run:")
			fmt.Println("  kubefirst beta vultr --help")

			if progress.Progress != nil {
				progress.Progress.Quit()
			}
		},
	}

	// on error, doesnt show helper/usage
	vultrCmd.SilenceUsage = true

	// wire up new commands
	vultrCmd.AddCommand(Create(), Destroy(), RootCredentials())

	return vultrCmd
}

func Create() *cobra.Command {
	createCmd := &cobra.Command{
		Use:              "create",
		Short:            "Create the Kubefirst platform running on Vultr Kubernetes",
		TraverseChildren: true,
		RunE:             createVultr,
		// PreRun:           common.CheckDocker,
	}

	vultrDefaults := constants.GetCloudDefaults().Vultr

	// todo review defaults and update descriptions
	createCmd.Flags().String("alerts-email", "", "Email address for Let's Encrypt certificate notifications (required)")
	createCmd.MarkFlagRequired("alerts-email")
	createCmd.Flags().Bool("ci", false, "If running Kubefirst in CI, set this flag to disable interactive features")
	createCmd.Flags().String("cloud-region", "ewr", "The Vultr region to provision infrastructure in")
	createCmd.Flags().String("cluster-name", "kubefirst", "The name of the cluster to create")
	createCmd.Flags().String("cluster-type", "mgmt", "The type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().String("node-count", vultrDefaults.NodeCount, "The node count for the cluster")
	createCmd.Flags().String("node-type", vultrDefaults.InstanceSize, "The instance size of the cluster to create")
	createCmd.Flags().String("dns-provider", "vultr", fmt.Sprintf("The DNS provider - one of: %s", supportedDNSProviders))
	createCmd.Flags().String("subdomain", "", "The subdomain to use for DNS records (Cloudflare)")
	createCmd.Flags().String("domain-name", "", "The Vultr DNS name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com) (required)")
	createCmd.MarkFlagRequired("domain-name")
	createCmd.Flags().String("git-provider", "github", fmt.Sprintf("The Git provider - one of: %s", supportedGitProviders))
	createCmd.Flags().String("git-protocol", "ssh", fmt.Sprintf("The Git protocol - one of: %s", supportedGitProtocolOverride))
	createCmd.Flags().String("github-org", "", "The GitHub organization for the new GitOps and metaphor repositories - required if using GitHub")
	createCmd.Flags().String("gitlab-group", "", "The GitLab group for the new GitOps and metaphor projects - required if using GitLab")
	createCmd.Flags().String("gitops-template-branch", "", "The branch to clone for the GitOps template repository")
	createCmd.Flags().String("gitops-template-url", "https://github.com/konstructio/gitops-template.git", "The fully qualified URL to the GitOps template repository to clone")
	createCmd.Flags().String("install-catalog-apps", "", "Comma separated values to install after provision")
	createCmd.Flags().Bool("use-telemetry", true, "Whether to emit telemetry")
	createCmd.Flags().Bool("install-kubefirst-pro", true, "Whether or not to install Kubefirst Pro")

	return createCmd
}

func Destroy() *cobra.Command {
	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "Destroy the Kubefirst platform",
		Long:  "Destroy the Kubefirst platform running in Vultr and remove all resources",
		RunE:  common.Destroy,
		// PreRun: common.CheckDocker,
	}

	return destroyCmd
}

func RootCredentials() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "root-credentials",
		Short: "Retrieve root authentication information for platform components",
		Long:  "Retrieve root authentication information for platform components",
		RunE:  common.GetRootCredentials,
	}

	authCmd.Flags().Bool("argocd", false, "Copy the ArgoCD password to the clipboard (optional)")
	authCmd.Flags().Bool("kbot", false, "Copy the kbot password to the clipboard (optional)")
	authCmd.Flags().Bool("vault", false, "Copy the vault password to the clipboard (optional)")

	return authCmd
}
