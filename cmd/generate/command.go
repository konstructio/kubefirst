/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package generate

import (
	"github.com/spf13/cobra"
)

var (
	// Create
	alertsEmailFlag          string
	ciFlag                   bool
	cloudRegionFlag          string
	clusterNameFlag          string
	clusterTypeFlag          string
	dnsProviderFlag          string
	domainNameFlag           string
	githubOrgFlag            string
	gitlabGroupFlag          string
	gitProviderFlag          string
	gitProtocolFlag          string
	gitopsTemplateURLFlag    string
	gitopsTemplateBranchFlag string
	useTelemetryFlag         bool

	// RootCredentials
	copyArgoCDPasswordToClipboardFlag bool
	copyKbotPasswordToClipboardFlag   bool
	copyVaultPasswordToClipboardFlag  bool

	// Supported providers
	supportedDNSProviders = []string{"vultr", "cloudflare"}
	supportedGitProviders = []string{"github", "gitlab"}
	// Supported git protocols
	supportedGitProtocolOverride = []string{"https", "ssh"}
)

func NewCommand() *cobra.Command {

	generateCmd := &cobra.Command{
		Use:   "vultr",
		Short: "kubefirst Vultr installation",
		Long:  "kubefirst vultr",
	}

	// on error, doesnt show helper/usage
	generateCmd.SilenceUsage = true

	// wire up new commands
	generateCmd.AddCommand(Generate())

	return generateCmd
}

func Generate() *cobra.Command {
	createCmd := &cobra.Command{
		Use:              "generate",
		Short:            "generate cluster content",
		TraverseChildren: true,
		RunE:             generate,
	}

	createCmd.Flags().BoolVar(&useTelemetryFlag, "use-telemetry", true, "whether to emit telemetry")

	return createCmd
}
