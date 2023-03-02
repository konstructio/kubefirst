package reports

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/kubefirst/kubefirst/internal/civo"
)

// CivoHandoffScreen prints the handoff screen
func CivoHandoffScreen(argocdAdminPassword, clusterName, domainName string, gitOwner string, config *civo.CivoConfig, dryRun bool, silentMode bool) {
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
	handOffData.WriteString(fmt.Sprintf("\n username: %s", "admin"))
	handOffData.WriteString(fmt.Sprintf("\n password: %s", argocdAdminPassword))

	// handOffData.WriteString("\n--- Argo Workflows ")
	// handOffData.WriteString(strings.Repeat("-", 51))
	// handOffData.WriteString(fmt.Sprintf("\n URL: %s", k3d.ArgoWorkflowsURL))

	// handOffData.WriteString("\n--- Atlantis ")
	// handOffData.WriteString(strings.Repeat("-", 57))
	// handOffData.WriteString(fmt.Sprintf("\n URL: %s", k3d.AtlantisURL))

	// handOffData.WriteString("\n--- Chartmuseum ")
	// handOffData.WriteString(strings.Repeat("-", 54))
	// handOffData.WriteString(fmt.Sprintf("\n URL: %s", k3d.ChartMuseumURL))

	// handOffData.WriteString("\n--- Metaphor ")
	// handOffData.WriteString(strings.Repeat("-", 57))
	// handOffData.WriteString("\n URLs: ")
	// handOffData.WriteString(fmt.Sprintf("\n  %s", k3d.MetaphorDevelopmentURL))
	// handOffData.WriteString(fmt.Sprintf("\n  %s", k3d.MetaphorStagingURL))
	// handOffData.WriteString(fmt.Sprintf("\n  %s", k3d.MetaphorProductionURL))

	handOffData.WriteString("\n--- Vault ")
	handOffData.WriteString(strings.Repeat("-", 60))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", fmt.Sprintf("https://vault.%s", domainName)))
	handOffData.WriteString(fmt.Sprintf("\n Root token: %s", "Check the secret vault-unseal-secret in Namespace vault"))
	handOffData.WriteString("\n" + strings.Repeat("-", 70))

	CommandSummary(handOffData)
}
