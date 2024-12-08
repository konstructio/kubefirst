/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

type CliFlags struct {
	AlertsEmail          string
	Ci                   bool
	CloudRegion          string
	CloudProvider        string
	ClusterName          string
	ClusterType          string
	DNSProvider          string
	DNSAzureRG           string
	DomainName           string
	SubDomainName        string
	GitProvider          string
	GitProtocol          string
	GithubOrg            string
	GitlabGroup          string
	GitopsTemplateBranch string
	GitopsTemplateURL    string
	GoogleProject        string
	UseTelemetry         bool
	ECR                  bool
	NodeType             string
	NodeCount            string
	InstallCatalogApps   string
	K3sSSHUser           string
	K3sSSHPrivateKey     string
	K3sServersPrivateIPs []string
	K3sServersPublicIPs  []string
	K3sServersArgs       []string
	InstallKubefirstPro  bool
	AmiType              string
}
