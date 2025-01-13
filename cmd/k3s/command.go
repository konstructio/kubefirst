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
	createCmd.Flags().String("alerts-email", "", "email address for let's encrypt certificate notifications (required)")
	createCmd.MarkFlagRequired("alerts-email")
	createCmd.Flags().Bool("ci", false, "if running kubefirst in ci, set this flag to disable interactive features")
	createCmd.Flags().String("cloud-region", "on-premise", "NOT USED, PRESENT FOR COMPATIBILITY ISSUE")
	createCmd.Flags().String("node-type", "on-premise", "NOT USED, PRESENT FOR COMPATIBILITY ISSUE")
	createCmd.Flags().String("node-count", "3", "NOT USED, PRESENT FOR COMPATIBILITY ISSUE")
	createCmd.Flags().String("cluster-name", "kubefirst", "the name of the cluster to create")
	createCmd.Flags().String("cluster-type", "mgmt", "the type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().StringSlice("servers-private-ips", []string{}, "the list of k3s (servers) private ip x.x.x.x,y.y.y.y comma separated  (required)")
	createCmd.MarkFlagRequired("servers-private-ips")
	createCmd.Flags().StringSlice("servers-public-ips", []string{}, "the list of k3s (servers) public ip x.x.x.x,y.y.y.y comma separated  (required)")
	createCmd.Flags().StringSlice("servers-args", []string{"--disable traefik", "--write-kubeconfig-mode 644"}, "list of k3s extras args to add to the k3s server installation,comma separated in between quote, if --servers-public-ips <VALUES> --tls-san <VALUES> is added to default --servers-args")
	createCmd.Flags().String("ssh-user", "root", "the user used to log into servers with ssh connection")
	createCmd.Flags().String("ssh-privatekey", "", "the private key used to log into servers with ssh connection")
	createCmd.MarkFlagRequired("ssh-privatekey")
	createCmd.Flags().String("dns-provider", "cloudflare", fmt.Sprintf("the dns provider - one of: %q", supportedDNSProviders))
	createCmd.Flags().String("subdomain", "", "the subdomain to use for DNS records (Cloudflare)")
	createCmd.Flags().String("domain-name", "", "the cloudProvider DNS Name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com) (required)")
	createCmd.Flags().String("git-provider", "github", fmt.Sprintf("the git provider - one of: %q", supportedGitProviders))
	createCmd.Flags().String("git-protocol", "ssh", fmt.Sprintf("the git protocol - one of: %q", supportedGitProtocolOverride))
	createCmd.Flags().String("github-org", "", "the GitHub organization for the new gitops and metaphor repositories - required if using github")
	createCmd.Flags().String("gitlab-group", "", "the GitLab group for the new gitops and metaphor projects - required if using gitlab")
	createCmd.Flags().String("gitops-template-branch", "", "the branch to clone for the gitops-template repository")
	createCmd.Flags().String("gitops-template-url", "https://github.com/konstructio/gitops-template.git", "the fully qualified url to the gitops-template repository to clone")
	createCmd.Flags().String("install-catalog-apps", "", "comma separated values to install after provision")
	createCmd.Flags().Bool("use-telemetry", true, "whether to emit telemetry")
	createCmd.Flags().Bool("force-destroy", false, "allows force destruction on objects (helpful for test environments, defaults to false)")
	createCmd.Flags().Bool("install-kubefirst-pro", true, "whether or not to install kubefirst pro")

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

	authCmd.Flags().Bool("argocd", false, "copy the ArgoCD password to the clipboard (optional)")
	authCmd.Flags().Bool("kbot", false, "copy the kbot password to the clipboard (optional)")
	authCmd.Flags().Bool("vault", false, "copy the vault password to the clipboard (optional)")

	return authCmd
}
