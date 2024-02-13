/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"

	"github.com/kubefirst/kubefirst/internal/progress"
	"github.com/spf13/cobra"
)

var (
	// Create
	applicationNameFlag      string
	applicationNamespaceFlag string
	ciFlag                   bool
	cloudRegionFlag          string
	clusterNameFlag          string
	clusterTypeFlag          string
	githubUserFlag           string
	githubOrgFlag            string
	gitlabGroupFlag          string
	gitProviderFlag          string
	gitProtocolFlag          string
	gitopsTemplateURLFlag    string
	gitopsTemplateBranchFlag string
	useTelemetryFlag         bool
	installCatalogApps       string

	// RootCredentials
	copyArgoCDPasswordToClipboardFlag bool
	copyKbotPasswordToClipboardFlag   bool
	copyVaultPasswordToClipboardFlag  bool

	// Supported git providers
	supportedGitProviders = []string{"github", "gitlab"}

	// Supported git providers
	supportedGitProtocolOverride = []string{"https", "ssh"}
)

func NewCommand() *cobra.Command {
	k3dCmd := &cobra.Command{
		Use:   "k3d",
		Short: "kubefirst k3d installation",
		Long:  "kubefirst k3d",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("To learn more about k3d in kubefirst, run:")
			fmt.Println("  kubefirst k3d --help")

			if progress.Progress != nil {
				progress.Progress.Quit()
			}
		},
	}

	// wire up new commands
	k3dCmd.AddCommand(Create(), Destroy(), MkCert(), RootCredentials(), UnsealVault())

	return k3dCmd
}

func LocalCommandAlias() *cobra.Command {
	localCmd := &cobra.Command{
		Use:   "local",
		Short: "kubefirst local installation with k3d",
		Long:  "kubefirst local installation with k3d",
	}

	// wire up new commands
	localCmd.AddCommand(Create(), Destroy(), MkCert(), RootCredentials(), UnsealVault())

	return localCmd
}

func Create() *cobra.Command {
	createCmd := &cobra.Command{
		Use:              "create",
		Short:            "create the kubefirst platform running in k3d on your localhost",
		TraverseChildren: true,
		RunE:             runK3d,
	}

	// todo review defaults and update descriptions
	createCmd.Flags().BoolVar(&ciFlag, "ci", false, "if running kubefirst in ci, set this flag to disable interactive features")
	createCmd.Flags().StringVar(&clusterNameFlag, "cluster-name", "kubefirst", "the name of the cluster to create")
	createCmd.Flags().StringVar(&clusterTypeFlag, "cluster-type", "mgmt", "the type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().StringVar(&gitProviderFlag, "git-provider", "github", fmt.Sprintf("the git provider - one of: %s", supportedGitProviders))
	createCmd.Flags().StringVar(&gitProtocolFlag, "git-protocol", "ssh", fmt.Sprintf("the git protocol - one of: %s", supportedGitProtocolOverride))
	createCmd.Flags().StringVar(&githubUserFlag, "github-user", "", "the GitHub user for the new gitops and metaphor repositories - this cannot be used with --github-org")
	createCmd.Flags().StringVar(&githubOrgFlag, "github-org", "", "the GitHub organization for the new gitops and metaphor repositories - this cannot be used with --github-user")
	createCmd.Flags().StringVar(&gitlabGroupFlag, "gitlab-group", "", "the GitLab group for the new gitops and metaphor projects - required if using gitlab")
	createCmd.Flags().StringVar(&gitopsTemplateBranchFlag, "gitops-template-branch", "", "the branch to clone for the gitops-template repository")
	createCmd.Flags().StringVar(&gitopsTemplateURLFlag, "gitops-template-url", "https://github.com/kubefirst/gitops-template.git", "the fully qualified url to the gitops-template repository to clone")
	createCmd.Flags().StringVar(&installCatalogApps, "install-catalog-apps", "", "comma seperated values of catalog apps to install after provision")
	createCmd.Flags().BoolVar(&useTelemetryFlag, "use-telemetry", true, "whether to emit telemetry")

	return createCmd
}

func Destroy() *cobra.Command {
	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "destroy the kubefirst platform",
		Long:  "deletes the GitHub resources, k3d resources, and local content to re-provision",
		RunE:  destroyK3d,
	}

	return destroyCmd
}

func MkCert() *cobra.Command {
	mkCertCmd := &cobra.Command{
		Use:   "mkcert",
		Short: "create a single ssl certificate for a local application",
		Long:  "create a single ssl certificate for a local application",
		RunE:  mkCert,
	}

	mkCertCmd.Flags().StringVar(&applicationNameFlag, "application", "", "the name of the application (required)")
	mkCertCmd.MarkFlagRequired("application")
	mkCertCmd.Flags().StringVar(&applicationNamespaceFlag, "namespace", "", "the application namespace (required)")
	mkCertCmd.MarkFlagRequired("namespace")

	return mkCertCmd
}

func RootCredentials() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "root-credentials",
		Short: "retrieve root authentication information for platform components",
		Long:  "retrieve root authentication information for platform components",
		RunE:  getK3dRootCredentials,
	}

	authCmd.Flags().BoolVar(&copyArgoCDPasswordToClipboardFlag, "argocd", false, "copy the argocd password to the clipboard (optional)")
	authCmd.Flags().BoolVar(&copyKbotPasswordToClipboardFlag, "kbot", false, "copy the kbot password to the clipboard (optional)")
	authCmd.Flags().BoolVar(&copyVaultPasswordToClipboardFlag, "vault", false, "copy the vault password to the clipboard (optional)")

	return authCmd
}

func UnsealVault() *cobra.Command {
	unsealVaultCmd := &cobra.Command{
		Use:   "unseal-vault",
		Short: "check to see if an existing vault instance is sealed and, if so, unseal it",
		Long:  "check to see if an existing vault instance is sealed and, if so, unseal it",
		RunE:  unsealVault,
	}

	return unsealVaultCmd
}
