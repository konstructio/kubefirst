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
	var err error

	var (
		alertsEmailFlag, cloudRegionFlag, dnsProviderFlag, subdomainFlag, domainNameFlag      string
		nodeTypeFlag, nodeCountFlag, installCatalogAppsFlag, gitProviderFlag, gitProtocolFlag string
		gitopsTemplateURLFlag, gitopsTemplateBranchFlag, githubOrgFlag, gitlabGroupFlag       string
		installKubefirstProFlag                                                               bool
	)

	flags := map[string]*string{
		"cluster-name":           &cliFlags.ClusterName,
		"github-org":             &githubOrgFlag,
		"gitlab-group":           &gitlabGroupFlag,
		"git-provider":           &gitProviderFlag,
		"git-protocol":           &gitProtocolFlag,
		"gitops-template-url":    &gitopsTemplateURLFlag,
		"gitops-template-branch": &gitopsTemplateBranchFlag,
		"install-catalog-apps":   &installCatalogAppsFlag,
	}

	for flag, target := range flags {
		if *target, err = cmd.Flags().GetString(flag); err != nil {
			return cliFlags, fmt.Errorf("failed to get %s flag: %w", flag, err)
		}
	}

	githubOrgFlag = strings.ToLower(githubOrgFlag)
	gitlabGroupFlag = strings.ToLower(gitlabGroupFlag)

	if cloudProvider != "k3d" {
		cloudSpecificFlags := map[string]*string{
			"alerts-email": &alertsEmailFlag,
			"cloud-region": &cloudRegionFlag,
			"dns-provider": &dnsProviderFlag,
			"subdomain":    &subdomainFlag,
			"domain-name":  &domainNameFlag,
			"node-type":    &nodeTypeFlag,
			"node-count":   &nodeCountFlag,
		}

		for flag, target := range cloudSpecificFlags {
			if *target, err = cmd.Flags().GetString(flag); err != nil {
				return cliFlags, fmt.Errorf("failed to get %s flag: %w", flag, err)
			}
		}

		if installKubefirstProFlag, err = cmd.Flags().GetBool("install-kubefirst-pro"); err != nil {
			return cliFlags, fmt.Errorf("failed to get install-kubefirst-pro flag: %w", err)
		}
	}

	switch cloudProvider {
	case "aws":
		ecrFlag, err := cmd.Flags().GetBool("ecr")
		if err != nil {
			return cliFlags, fmt.Errorf("failed to get ecr flag: %w", err)
		}

		cliFlags.ECR = ecrFlag

		amiType, err := cmd.Flags().GetString("ami-type")
		if err != nil {
			return cliFlags, fmt.Errorf("failed to get ami type: %w", err)
		}
		cliFlags.AMIType = amiType

	case "azure":
		dnsAzureResourceGroup, err := cmd.Flags().GetString("dns-azure-resource-group")
		if err != nil {
			return cliFlags, fmt.Errorf("failed to get dns-azure-resource-group flag: %w", err)
		}
		cliFlags.DNSAzureRG = dnsAzureResourceGroup

	case "google":
		googleProject, err := cmd.Flags().GetString("google-project")
		if err != nil {
			return cliFlags, fmt.Errorf("failed to get google-project flag: %w", err)
		}

		cliFlags.GoogleProject = googleProject

	case "k3s":
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

	// Assign collected values to cliFlags
	cliFlags = types.CliFlags{
		AlertsEmail:          alertsEmailFlag,
		CloudRegion:          cloudRegionFlag,
		ClusterName:          cliFlags.ClusterName,
		DNSProvider:          dnsProviderFlag,
		SubDomainName:        subdomainFlag,
		DomainName:           domainNameFlag,
		GitProtocol:          gitProtocolFlag,
		GitProvider:          gitProviderFlag,
		GithubOrg:            githubOrgFlag,
		GitlabGroup:          gitlabGroupFlag,
		GitopsTemplateBranch: gitopsTemplateBranchFlag,
		GitopsTemplateURL:    gitopsTemplateURLFlag,
		UseTelemetry:         cliFlags.UseTelemetry,
		CloudProvider:        cloudProvider,
		NodeType:             nodeTypeFlag,
		NodeCount:            nodeCountFlag,
		InstallCatalogApps:   installCatalogAppsFlag,
		InstallKubefirstPro:  installKubefirstProFlag,
	}

	// Set Viper configurations
	viperConfigs := map[string]interface{}{
		"flags.alerts-email":       cliFlags.AlertsEmail,
		"flags.cluster-name":       cliFlags.ClusterName,
		"flags.dns-provider":       cliFlags.DNSProvider,
		"flags.domain-name":        cliFlags.DomainName,
		"flags.git-provider":       cliFlags.GitProvider,
		"flags.git-protocol":       cliFlags.GitProtocol,
		"flags.cloud-region":       cliFlags.CloudRegion,
		"kubefirst.cloud-provider": cloudProvider,
	}

	for key, value := range viperConfigs {
		viper.Set(key, value)
	}

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
