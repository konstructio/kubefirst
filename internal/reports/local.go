/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package reports

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/kubefirst/kubefirst/internal/k3d"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var caser = cases.Title(language.AmericanEnglish)

// LocalHandoffScreenV2 prints the handoff screen
func LocalHandoffScreenV2(argocdAdminPassword, clusterName, gitDestDescriptor string, gitOwner string, config *k3d.K3dConfig, dryRun bool, silentMode bool) {
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

	handOffData.WriteString("\n\nNote:")
	handOffData.WriteString("\n  Kubefirst generated certificates to ensure secure connections to")
	handOffData.WriteString("\n  your local kubernetes services. However they will not be")
	handOffData.WriteString("\n  trusted by your browser by default.")
	handOffData.WriteString("\n")
	handOffData.WriteString("\n  It is safe to ignore the warning and continue to these sites, or ")
	handOffData.WriteString("\n  to remove these warnings, you can install a new certificate ")
	handOffData.WriteString("\n  to your local trust store by running the following command: ")
	handOffData.WriteString(fmt.Sprintf("\n"+"\n    %s -install"+"\n", config.MkCertClient))
	handOffData.WriteString("\n  For more details on the mkcert utility, please see:")
	handOffData.WriteString("\n  https://github.com/FiloSottile/mkcert")

	handOffData.WriteString(fmt.Sprintf("\n\n--- %s ", caser.String(config.GitProvider)))
	handOffData.WriteString(strings.Repeat("-", 59))
	handOffData.WriteString(fmt.Sprintf("\n %s: %s", caser.String(gitDestDescriptor), gitOwner))
	handOffData.WriteString("\n Repositories: ")
	handOffData.WriteString(fmt.Sprintf("\n  %s", config.DestinationGitopsRepoHttpsURL))
	handOffData.WriteString(fmt.Sprintf("\n  %s", config.DestinationMetaphorRepoHttpsURL))

	handOffData.WriteString("\n--- Kubefirst Console ")
	handOffData.WriteString(strings.Repeat("-", 48))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", k3d.KubefirstConsoleURL))

	handOffData.WriteString("\n--- ArgoCD ")
	handOffData.WriteString(strings.Repeat("-", 59))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", k3d.ArgocdURL))

	handOffData.WriteString("\n--- Vault ")
	handOffData.WriteString(strings.Repeat("-", 60))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", k3d.VaultURL))
	handOffData.WriteString("\n" + strings.Repeat("-", 70))

	handOffData.WriteString("\n\nNote:")
	handOffData.WriteString("\n  To retrieve root credentials for your kubefirst platform, including")
	handOffData.WriteString("\n  ArgoCD, the kbot user password, and Vault, run the following command:")
	handOffData.WriteString(fmt.Sprintf("\n"+"\n    kubefirst %s root-credentials"+"\n", k3d.CloudProvider))
	handOffData.WriteString("\n  Note that this command allows you to copy these passwords diretly")
	handOffData.WriteString("\n  to your clipboard. Provide the -h flag for additional details.")

	CommandSummary(handOffData)
}
