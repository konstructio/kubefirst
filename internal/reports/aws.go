/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package reports

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	awsinternal "github.com/kubefirst/kubefirst/internal/aws"
)

// AwsHandoffScreen prints the handoff screen
func AwsHandoffScreen(argocdAdminPassword, clusterName, domainName string, gitOwner string, config *awsinternal.AwsConfig, dryRun bool, silentMode bool) {
	// prepare data for the handoff report
	if dryRun {
		log.Printf("[#99] Dry-run mode, LocalHandoffScreen skipped.")
		return
	}

	if silentMode {
		log.Printf("[#99] Silent mode enabled, LocalHandoffScreen skipped, please check ~/.kubefirst file for your cluster and service credentials.")
		return
	}

	var handOffData bytes.Buffer

	handOffData.WriteString(strings.Repeat("-", 70))
	handOffData.WriteString("\n			!!! THIS TEXT BOX SCROLLS (use arrow keys) !!!")

	handOffData.WriteString(fmt.Sprintf("\n\nCluster %q is up and running!:", clusterName))
	handOffData.WriteString("\nThis information is available at $HOME/.kubefirst ")
	handOffData.WriteString("\n")
	handOffData.WriteString("\nPress ESC to leave this screen and return to your shell.")

	handOffData.WriteString(fmt.Sprintf("\n\n--- %s ", caser.String(config.GitProvider)))
	handOffData.WriteString(strings.Repeat("-", 59))
	handOffData.WriteString(fmt.Sprintf("\n Owner: %s", gitOwner))
	handOffData.WriteString("\n Repos: ")
	handOffData.WriteString(fmt.Sprintf("\n  %s", config.DestinationGitopsRepoHttpsURL))
	handOffData.WriteString(fmt.Sprintf("\n  %s", config.DestinationMetaphorRepoHttpsURL))

	handOffData.WriteString("\n--- Kubefirst Console ")
	handOffData.WriteString(strings.Repeat("-", 48))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", "http://localhost:9094/services"))

	handOffData.WriteString("\n--- ArgoCD ")
	handOffData.WriteString(strings.Repeat("-", 59))
	handOffData.WriteString(fmt.Sprintf("\n URL: https://argocd.%s", domainName))

	handOffData.WriteString("\n--- Vault ")
	handOffData.WriteString(strings.Repeat("-", 60))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", fmt.Sprintf("https://vault.%s", domainName)))
	handOffData.WriteString("\n" + strings.Repeat("-", 70))

	handOffData.WriteString("\n\nNote:")
	handOffData.WriteString("\n  To retrieve root credentials for your kubefirst platform, including")
	handOffData.WriteString("\n  ArgoCD, the kbot user password, and Vault, run the following command:")
	handOffData.WriteString(fmt.Sprintf("\n"+"\n    kubefirst %s root-credentials"+"\n", awsinternal.CloudProvider))
	handOffData.WriteString("\n  Note that this command allows you to copy these passwords diretly")
	handOffData.WriteString("\n  to your clipboard. Provide the -h flag for additional details.")

	CommandSummary(handOffData)
}
