/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Create
	cloudRegionFlag          string
	clusterNameFlag          string
	clusterTypeFlag          string
	githubUserFlag           string
	githubOrgFlag            string
	gitlabGroupFlag          string
	gitProviderFlag          string
	gitopsTemplateURLFlag    string
	gitopsTemplateBranchFlag string
	useTelemetryFlag         bool

	// RootCredentials
	copyArgoCDPasswordToClipboardFlag bool
	copyKbotPasswordToClipboardFlag   bool
	copyVaultPasswordToClipboardFlag  bool

	// Supported git providers
	supportedGitProviders = []string{"github", "gitlab"}
)

func NewCommand() *cobra.Command {
	k3dCmd := &cobra.Command{
		Use:   "k3d",
		Short: "kubefirst k3d installation",
		Long:  "kubefirst k3d",
	}

	// wire up new commands
	k3dCmd.AddCommand(Create(), Destroy(), RootCredentials(), UnsealVault())

	return k3dCmd
}

func LocalCommandAlias() *cobra.Command {

	localCmd := &cobra.Command{
		Use:   "local",
		Short: "kubefirst local installation with k3d",
		Long:  "kubefirst local installation with k3d",
	}

	// wire up new commands
	localCmd.AddCommand(Create(), Destroy(), RootCredentials())

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
	createCmd.Flags().StringVar(&clusterNameFlag, "cluster-name", "kubefirst", "the name of the cluster to create")
	createCmd.Flags().StringVar(&clusterTypeFlag, "cluster-type", "mgmt", "the type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().StringVar(&gitProviderFlag, "git-provider", "github", fmt.Sprintf("the git provider - one of: %s", supportedGitProviders))
	createCmd.Flags().StringVar(&githubUserFlag, "github-user", "", "the GitHub user for the new gitops and metaphor repositories - this cannot be used with --github-org")
	createCmd.Flags().StringVar(&githubOrgFlag, "github-org", "", "the GitHub organization for the new gitops and metaphor repositories - this cannot be used with --github-user")
	createCmd.Flags().StringVar(&gitlabGroupFlag, "gitlab-group", "", "the GitLab group for the new gitops and metaphor projects - required if using gitlab")
	createCmd.Flags().StringVar(&gitopsTemplateBranchFlag, "gitops-template-branch", "", "the branch to clone for the gitops-template repository")
	createCmd.Flags().StringVar(&gitopsTemplateURLFlag, "gitops-template-url", "https://github.com/kubefirst/gitops-template.git", "the fully qualified url to the gitops-template repository to clone")
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
