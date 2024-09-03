/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package utilities

import (
	"fmt"
	"strings"

	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/konstructio/kubefirst/internal/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func GetFlags(cmd *cobra.Command, cloudProvider string) (types.CliFlags, error) {
	cliFlags := types.CliFlags{}

	alertsEmailFlag, err := cmd.Flags().GetString("alerts-email")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, fmt.Errorf("failed to get alerts-email flag: %w", err)
	}

	cloudRegionFlag, err := cmd.Flags().GetString("cloud-region")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, fmt.Errorf("failed to get cloud-region flag: %w", err)
	}

	clusterNameFlag, err := cmd.Flags().GetString("cluster-name")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, fmt.Errorf("failed to get cluster-name flag: %w", err)
	}

	dnsProviderFlag, err := cmd.Flags().GetString("dns-provider")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, fmt.Errorf("failed to get dns-provider flag: %w", err)
	}

	subdomainFlag, err := cmd.Flags().GetString("subdomain")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, fmt.Errorf("failed to get subdomain flag: %w", err)
	}

	domainNameFlag, err := cmd.Flags().GetString("domain-name")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, fmt.Errorf("failed to get domain-name flag: %w", err)
	}

	githubOrgFlag, err := cmd.Flags().GetString("github-org")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, fmt.Errorf("failed to get github-org flag: %w", err)
	}
	githubOrgFlag = strings.ToLower(githubOrgFlag)

	gitlabGroupFlag, err := cmd.Flags().GetString("gitlab-group")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, fmt.Errorf("failed to get gitlab-group flag: %w", err)
	}
	gitlabGroupFlag = strings.ToLower(gitlabGroupFlag)

	gitProviderFlag, err := cmd.Flags().GetString("git-provider")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, fmt.Errorf("failed to get git-provider flag: %w", err)
	}

	gitProtocolFlag, err := cmd.Flags().GetString("git-protocol")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, fmt.Errorf("failed to get git-protocol flag: %w", err)
	}

	gitopsTemplateURLFlag, err := cmd.Flags().GetString("gitops-template-url")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, fmt.Errorf("failed to get gitops-template-url flag: %w", err)
	}

	gitopsTemplateBranchFlag, err := cmd.Flags().GetString("gitops-template-branch")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, fmt.Errorf("failed to get gitops-template-branch flag: %w", err)
	}

	useTelemetryFlag, err := cmd.Flags().GetBool("use-telemetry")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, fmt.Errorf("failed to get use-telemetry flag: %w", err)
	}

	nodeTypeFlag, err := cmd.Flags().GetString("node-type")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, fmt.Errorf("failed to get node-type flag: %w", err)
	}

	installCatalogAppsFlag, err := cmd.Flags().GetString("install-catalog-apps")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, fmt.Errorf("failed to get install-catalog-apps flag: %w", err)
	}

	nodeCountFlag, err := cmd.Flags().GetString("node-count")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, fmt.Errorf("failed to get node-count flag: %w", err)
	}

	installKubefirstProFlag, err := cmd.Flags().GetBool("install-kubefirst-pro")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, fmt.Errorf("failed to get install-kubefirst-pro flag: %w", err)
	}

	if cloudProvider == "aws" {
		ecrFlag, err := cmd.Flags().GetBool("ecr")
		if err != nil {
			progress.Error(err.Error())
			return cliFlags, fmt.Errorf("failed to get ecr flag: %w", err)
		}

		cliFlags.ECR = ecrFlag
	}

	if cloudProvider == "google" {
		googleProject, err := cmd.Flags().GetString("google-project")
		if err != nil {
			progress.Error(err.Error())
			return cliFlags, fmt.Errorf("failed to get google-project flag: %w", err)
		}

		cliFlags.GoogleProject = googleProject
	}

	if cloudProvider == "k3s" {
		k3sServersPrivateIps, err := cmd.Flags().GetStringSlice("servers-private-ips")
		if err != nil {
			progress.Error(err.Error())
			return cliFlags, fmt.Errorf("failed to get servers-private-ips flag: %w", err)
		}
		cliFlags.K3sServersPrivateIPs = k3sServersPrivateIps

		k3sServersPublicIps, err := cmd.Flags().GetStringSlice("servers-public-ips")
		if err != nil {
			progress.Error(err.Error())
			return cliFlags, fmt.Errorf("failed to get servers-public-ips flag: %w", err)
		}
		cliFlags.K3sServersPublicIPs = k3sServersPublicIps

		k3sSSHUserFlag, err := cmd.Flags().GetString("ssh-user")
		if err != nil {
			progress.Error(err.Error())
			return cliFlags, fmt.Errorf("failed to get ssh-user flag: %w", err)
		}
		cliFlags.K3sSSHUser = k3sSSHUserFlag

		k3sSSHPrivateKeyFlag, err := cmd.Flags().GetString("ssh-privatekey")
		if err != nil {
			progress.Error(err.Error())
			return cliFlags, fmt.Errorf("failed to get ssh-privatekey flag: %w", err)
		}
		cliFlags.K3sSSHPrivateKey = k3sSSHPrivateKeyFlag

		K3sServersArgsFlags, err := cmd.Flags().GetStringSlice("servers-args")
		if err != nil {
			progress.Error(err.Error())
			return cliFlags, fmt.Errorf("failed to get servers-args flag: %w", err)
		}
		cliFlags.K3sServersArgs = K3sServersArgsFlags
	}

	cliFlags.AlertsEmail = alertsEmailFlag
	cliFlags.CloudRegion = cloudRegionFlag
	cliFlags.ClusterName = clusterNameFlag
	cliFlags.DNSProvider = dnsProviderFlag
	cliFlags.SubDomainName = subdomainFlag
	cliFlags.DomainName = domainNameFlag
	cliFlags.GitProtocol = gitProtocolFlag
	cliFlags.GitProvider = gitProviderFlag
	cliFlags.GithubOrg = githubOrgFlag
	cliFlags.GitlabGroup = gitlabGroupFlag
	cliFlags.GitopsTemplateBranch = gitopsTemplateBranchFlag
	cliFlags.GitopsTemplateURL = gitopsTemplateURLFlag
	cliFlags.UseTelemetry = useTelemetryFlag
	cliFlags.CloudProvider = cloudProvider
	cliFlags.NodeType = nodeTypeFlag
	cliFlags.NodeCount = nodeCountFlag
	cliFlags.InstallCatalogApps = installCatalogAppsFlag
	cliFlags.InstallKubefirstPro = installKubefirstProFlag

	viper.Set("flags.alerts-email", cliFlags.AlertsEmail)
	viper.Set("flags.cluster-name", cliFlags.ClusterName)
	viper.Set("flags.dns-provider", cliFlags.DNSProvider)
	viper.Set("flags.domain-name", cliFlags.DomainName)
	viper.Set("flags.git-provider", cliFlags.GitProvider)
	viper.Set("flags.git-protocol", cliFlags.GitProtocol)
	viper.Set("flags.cloud-region", cliFlags.CloudRegion)
	viper.Set("kubefirst.cloud-provider", cloudProvider)
	if cloudProvider == "k3s" {
		viper.Set("flags.servers-private-ips", cliFlags.K3sServersPrivateIPs)
		viper.Set("flags.servers-public-ips", cliFlags.K3sServersPublicIPs)
		viper.Set("flags.ssh-user", cliFlags.K3sSSHUser)
		viper.Set("flags.ssh-privatekey", cliFlags.K3sSSHPrivateKey)
		viper.Set("flags.servers-args", cliFlags.K3sServersArgs)
	}
	if err := viper.WriteConfig(); err != nil {
		return cliFlags, fmt.Errorf("failed to write configuration: %w", err)
	}

	return cliFlags, nil
}
