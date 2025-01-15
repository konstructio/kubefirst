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
	createCmd.Flags().String("alerts-email", "", "Email address for Let's Encrypt certificate notifications (required)")
	createCmd.MarkFlagRequired("alerts-email")
	createCmd.Flags().Bool("ci", false, "If running Kubefirst in CI, set this flag to disable interactive features")
	createCmd.Flags().String("cloud-region", "NYC1", "The Civo region to provision infrastructure in")
	createCmd.Flags().String("cluster-name", "kubefirst", "The name of the cluster to create")
	createCmd.Flags().String("cluster-type", "mgmt", "The type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().String("node-count", civoDefaults.NodeCount, "The node count for the cluster")
	createCmd.Flags().String("node-type", civoDefaults.InstanceSize, "The instance size of the cluster to create")
	createCmd.Flags().String("dns-provider", "civo", fmt.Sprintf("The DNS provider - one of: %s", supportedDNSProviders))
	createCmd.Flags().String("subdomain", "", "The subdomain to use for DNS records (Cloudflare)")
	createCmd.Flags().String("domain-name", "", "The Civo DNS Name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com) (required)")
	createCmd.MarkFlagRequired("domain-name")
	createCmd.Flags().String("git-provider", "github", fmt.Sprintf("The git provider - one of: %s", supportedGitProviders))
	createCmd.Flags().String("git-protocol", "ssh", fmt.Sprintf("The git protocol - one of: %s", supportedGitProtocolOverride))
	createCmd.Flags().String("github-org", "", "The GitHub organization for the new GitOps and Metaphor repositories - required if using GitHub")
	createCmd.Flags().String("gitlab-group", "", "The GitLab group for the new GitOps and Metaphor projects - required if using GitLab")
	createCmd.Flags().String("gitops-template-branch", "", "The branch to clone for the gitops-template repository")
	createCmd.Flags().String("gitops-template-url", "https://github.com/konstructio/gitops-template.git", "The fully qualified URL to the gitops-template repository to clone")
	createCmd.Flags().String("install-catalog-apps", "", "Comma separated values to install after provision")
	createCmd.Flags().Bool("use-telemetry", true, "Whether to emit telemetry")
	createCmd.Flags().Bool("install-kubefirst-pro", true, "Whether or not to install Kubefirst Pro")

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

	quotaCmd.Flags().String("cloud-region", "NYC1", "The Civo region to monitor quotas in")

	return quotaCmd
}

func RootCredentials() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "root-credentials",
		Short: "Retrieve root authentication information for platform components",
		Long:  "Retrieve root authentication information for platform components",
		RunE:  common.GetRootCredentials,
	}

	authCmd.Flags().Bool("argocd", false, "Copy the ArgoCD password to the clipboard (optional)")
	authCmd.Flags().Bool("kbot", false, "Copy the Kbot password to the clipboard (optional)")
	authCmd.Flags().Bool("vault", false, "Copy the Vault password to the clipboard (optional)")

	return authCmd
}
