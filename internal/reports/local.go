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
func LocalHandoffScreenV2(argocdAdminPassword, clusterName, gitOwner string, config *k3d.K3dConfig, dryRun bool, silentMode bool) {
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
	handOffData.WriteString("\n  https://github.com/FiloSottile/mkcert#changing-the-location-of-the-ca-files")

	handOffData.WriteString(fmt.Sprintf("\n\n--- %s ", caser.String(config.GitProvider)))
	handOffData.WriteString(strings.Repeat("-", 59))
	handOffData.WriteString(fmt.Sprintf("\n Owner: %s", gitOwner))
	handOffData.WriteString("\n Repos: ")
	handOffData.WriteString(fmt.Sprintf("\n  %s", config.DestinationGitopsRepoGitURL))
	handOffData.WriteString(fmt.Sprintf("\n  %s", config.DestinationMetaphorRepoGitURL))

	handOffData.WriteString("\n--- Kubefirst Console ")
	handOffData.WriteString(strings.Repeat("-", 48))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", k3d.KubefirstConsoleURL))

	handOffData.WriteString("\n--- ArgoCD ")
	handOffData.WriteString(strings.Repeat("-", 59))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", k3d.ArgocdURL))
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
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", k3d.VaultURL))
	handOffData.WriteString(fmt.Sprintf("\n Root token: %s", "k1_local_vault_token"))
	handOffData.WriteString("\n" + strings.Repeat("-", 70))

	CommandSummary(handOffData)

}
