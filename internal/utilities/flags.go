/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package utilities

import (
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
		return cliFlags, err
	}

	cloudRegionFlag, err := cmd.Flags().GetString("cloud-region")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, err
	}

	clusterNameFlag, err := cmd.Flags().GetString("cluster-name")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, err
	}

	dnsProviderFlag, err := cmd.Flags().GetString("dns-provider")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, err
	}

	subdomainFlag, err := cmd.Flags().GetString("subdomain")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, err
	}

	domainNameFlag, err := cmd.Flags().GetString("domain-name")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, err
	}

	githubOrgFlag, err := cmd.Flags().GetString("github-org")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, err
	}
	githubOrgFlag = strings.ToLower(githubOrgFlag)

	gitlabGroupFlag, err := cmd.Flags().GetString("gitlab-group")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, err
	}
	gitlabGroupFlag = strings.ToLower(gitlabGroupFlag)

	gitProviderFlag, err := cmd.Flags().GetString("git-provider")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, err
	}

	gitProtocolFlag, err := cmd.Flags().GetString("git-protocol")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, err
	}

	gitopsRepoNameFlag, err := cmd.Flags().GetString("gitopsRepoName")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags,err
	}

	metaphorRepoNameFlag, err := cmd.Flags().GetString("metaphorRepoName")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags,err
	}

	adminTeamNameFlag, err := cmd.Flags().GetString("adminTeamName")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags,err
	}

	developerTeamNameFlag, err := cmd.Flags().GetString("developerTeamName")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags,err
	}

	gitopsTemplateURLFlag, err := cmd.Flags().GetString("gitops-template-url")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, err
	}

	gitopsTemplateBranchFlag, err := cmd.Flags().GetString("gitops-template-branch")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, err
	}

	useTelemetryFlag, err := cmd.Flags().GetBool("use-telemetry")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, err
	}

	nodeTypeFlag, err := cmd.Flags().GetString("node-type")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, err
	}

	installCatalogAppsFlag, err := cmd.Flags().GetString("install-catalog-apps")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, err
	}

	nodeCountFlag, err := cmd.Flags().GetString("node-count")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, err
	}

	installKubefirstProFlag, err := cmd.Flags().GetBool("install-kubefirst-pro")
	if err != nil {
		progress.Error(err.Error())
		return cliFlags, err
	}

	if cloudProvider == "aws" {
		ecrFlag, err := cmd.Flags().GetBool("ecr")
		if err != nil {
			progress.Error(err.Error())
			return cliFlags, err
		}

		cliFlags.Ecr = ecrFlag
	}

	if cloudProvider == "google" {
		googleProject, err := cmd.Flags().GetString("google-project")
		if err != nil {
			progress.Error(err.Error())
			return cliFlags, err
		}

		cliFlags.GoogleProject = googleProject
	}

	// TODO: reafactor this part
	if cloudProvider == "k3s" {
		k3sServersPrivateIps, err := cmd.Flags().GetStringSlice("servers-private-ips")
		if err != nil {
			progress.Error(err.Error())
			return cliFlags, err
		}
		cliFlags.K3sServersPrivateIps = k3sServersPrivateIps

		k3sServersPublicIps, err := cmd.Flags().GetStringSlice("servers-public-ips")
		if err != nil {
			progress.Error(err.Error())
			return cliFlags, err
		}
		cliFlags.K3sServersPublicIps = k3sServersPublicIps

		k3sSshUserFlag, err := cmd.Flags().GetString("ssh-user")
		if err != nil {
			progress.Error(err.Error())
			return cliFlags, err
		}
		cliFlags.K3sSshUser = k3sSshUserFlag

		k3sSshPrivateKeyFlag, err := cmd.Flags().GetString("ssh-privatekey")
		if err != nil {
			progress.Error(err.Error())
			return cliFlags, err
		}
		cliFlags.K3sSshPrivateKey = k3sSshPrivateKeyFlag

		K3sServersArgsFlags, err := cmd.Flags().GetStringSlice("servers-args")
		if err != nil {
			progress.Error(err.Error())
			return cliFlags, err
		}
		cliFlags.K3sServersArgs = K3sServersArgsFlags
	}

	cliFlags.AlertsEmail = alertsEmailFlag
	cliFlags.CloudRegion = cloudRegionFlag
	cliFlags.ClusterName = clusterNameFlag
	cliFlags.DnsProvider = dnsProviderFlag
	cliFlags.SubDomainName = subdomainFlag
	cliFlags.DomainName = domainNameFlag
	cliFlags.GitProtocol = gitProtocolFlag
	cliFlags.GitProvider = gitProviderFlag
	cliFlags.GithubOrg = githubOrgFlag
	cliFlags.GitlabGroup = gitlabGroupFlag
	cliFlags.GitopsTemplateBranch = gitopsTemplateBranchFlag
	cliFlags.GitopsTemplateURL = gitopsTemplateURLFlag
	cliFlags.UseTelemetry = useTelemetryFlag
	cliFlags.GitopsRepoName = gitopsRepoNameFlag
	cliFlags.MetaphorRepoName = metaphorRepoNameFlag
	cliFlags.AdminTeamName = adminTeamNameFlag
	cliFlags.DeveloperTeamName = developerTeamNameFlag
	cliFlags.CloudProvider = cloudProvider
	cliFlags.NodeType = nodeTypeFlag
	cliFlags.NodeCount = nodeCountFlag
	cliFlags.InstallCatalogApps = installCatalogAppsFlag
	cliFlags.InstallKubefirstPro = installKubefirstProFlag

	viper.Set("flags.alerts-email", cliFlags.AlertsEmail)
	viper.Set("flags.cluster-name", cliFlags.ClusterName)
	viper.Set("flags.dns-provider", cliFlags.DnsProvider)
	viper.Set("flags.domain-name", cliFlags.DomainName)
	viper.Set("flags.git-provider", cliFlags.GitProvider)
	viper.Set("flags.git-protocol", cliFlags.GitProtocol)
	viper.Set("flags.cloud-region", cliFlags.CloudRegion)
	viper.Set("kubefirst.cloud-provider", cloudProvider)
	if cloudProvider == "k3s" {
		viper.Set("flags.servers-private-ips", cliFlags.K3sServersPrivateIps)
		viper.Set("flags.servers-public-ips", cliFlags.K3sServersPublicIps)
		viper.Set("flags.ssh-user", cliFlags.K3sSshUser)
		viper.Set("flags.ssh-privatekey", cliFlags.K3sSshPrivateKey)
		viper.Set("flags.servers-args", cliFlags.K3sServersArgs)
	}
	viper.WriteConfig()

	return cliFlags, nil
}
