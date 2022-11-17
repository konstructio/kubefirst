package reports

import (
	"bytes"
	"fmt"
	"github.com/kubefirst/kubefirst/pkg"
	"log"
	"strings"

	"github.com/spf13/viper"
)

func PrintSectionRepoGithub() []byte {
	var handOffData bytes.Buffer

	// todo construct these urls upfront on init
	handOffData.WriteString("\n--- Github ")
	handOffData.WriteString(strings.Repeat("-", 59))
	handOffData.WriteString(fmt.Sprintf("\n owner: %s", viper.GetString("github.owner")))
	handOffData.WriteString("\n Repos: ")
	handOffData.WriteString(fmt.Sprintf("\n  %s", fmt.Sprintf("https://%s/%s/gitops", viper.GetString("github.host"), viper.GetString("github.owner"))))
	handOffData.WriteString(fmt.Sprintf("\n  %s", fmt.Sprintf("https://%s/%s/metaphor", viper.GetString("github.host"), viper.GetString("github.owner"))))

	return handOffData.Bytes()
}

func PrintSectionRepoGitlab() []byte {
	var handOffData bytes.Buffer

	handOffData.WriteString("\n--- GitLab ")
	handOffData.WriteString(strings.Repeat("-", 59))
	handOffData.WriteString(fmt.Sprintf("\n username: %s", "root"))
	handOffData.WriteString(fmt.Sprintf("\n password: %s", viper.GetString("gitlab.root.password")))
	handOffData.WriteString("\n Repos: ")
	handOffData.WriteString(fmt.Sprintf("\n  %s", fmt.Sprintf("https://gitlab.%s/kubefirst/gitops", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(fmt.Sprintf("\n  %s", fmt.Sprintf("https://gitlab.%s/kubefirst/metaphor", viper.GetString("aws.hostedzonename"))))

	return handOffData.Bytes()
}

func PrintSectionOverview() []byte {
	var handOffData bytes.Buffer
	handOffData.WriteString(strings.Repeat("-", 70))
	handOffData.WriteString(fmt.Sprintf("\nCluster %q is up and running!:", viper.GetString("cluster-name")))
	handOffData.WriteString("\nThis information is available at $HOME/.kubefirst ")
	handOffData.WriteString("\n\nAccess the kubefirst-console from your browser at:\n http://localhost:9094\n")
	handOffData.WriteString("\nPress ESC to leave this screen and return to your shell.")

	return handOffData.Bytes()
}

func PrintSectionAws() []byte {
	var handOffData bytes.Buffer
	handOffData.WriteString("\n--- AWS ")
	handOffData.WriteString(strings.Repeat("-", 62))
	handOffData.WriteString(fmt.Sprintf("\n AWS Account Id: %s", viper.GetString("aws.accountid")))
	handOffData.WriteString(fmt.Sprintf("\n AWS hosted zone name: %s", viper.GetString("aws.hostedzonename")))
	handOffData.WriteString(fmt.Sprintf("\n AWS region: %s", viper.GetString("aws.region")))
	return handOffData.Bytes()
}

func PrintSectionVault() []byte {

	var vaultURL string
	if viper.GetString("cloud") == pkg.CloudK3d {
		vaultURL = "http://localhost:8200"
	} else {
		vaultURL = fmt.Sprintf("https://vault.%s", viper.GetString("aws.hostedzonename"))
	}

	var handOffData bytes.Buffer
	handOffData.WriteString("\n--- Vault ")
	handOffData.WriteString(strings.Repeat("-", 60))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", vaultURL))
	handOffData.WriteString(fmt.Sprintf("\n token: %s", viper.GetString("vault.token")))
	return handOffData.Bytes()
}

func PrintSectionArgoCD() []byte {

	var argoCdURL string
	if viper.GetString("cloud") == pkg.CloudK3d {
		argoCdURL = "http://localhost:8080"
	} else {
		argoCdURL = fmt.Sprintf("https://argocd.%s", viper.GetString("aws.hostedzonename"))
	}

	var handOffData bytes.Buffer
	handOffData.WriteString("\n--- ArgoCD ")
	handOffData.WriteString(strings.Repeat("-", 59))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", argoCdURL))
	handOffData.WriteString(fmt.Sprintf("\n username: %s", viper.GetString("argocd.admin.username")))
	handOffData.WriteString(fmt.Sprintf("\n password: %s", viper.GetString("argocd.admin.password")))

	return handOffData.Bytes()
}

func PrintSectionArgoWorkflows() []byte {

	var argoWorkflowsURL string
	if viper.GetString("cloud") == pkg.CloudK3d {
		argoWorkflowsURL = "http://localhost:2746"
	} else {
		argoWorkflowsURL = fmt.Sprintf("https://argo.%s", viper.GetString("aws.hostedzonename"))
	}

	var handOffData bytes.Buffer
	handOffData.WriteString("\n--- Argo Workflows ")
	handOffData.WriteString(strings.Repeat("-", 51))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", argoWorkflowsURL))

	if viper.GetString("cloud") == pkg.CloudK3d {
		return handOffData.Bytes()
	} else {
		handOffData.WriteString("\n sso credentials only ")
		handOffData.WriteString("\n * sso enabled ")

		return handOffData.Bytes()
	}
}

func PrintSectionAtlantis() []byte {

	var atlantisUrl string
	if viper.GetString("cloud") == pkg.CloudK3d {
		atlantisUrl = "http://localhost:4141"
	} else {
		atlantisUrl = fmt.Sprintf("https://atlantis.%s", viper.GetString("aws.hostedzonename"))
	}

	var handOffData bytes.Buffer
	handOffData.WriteString("\n--- Atlantis ")
	handOffData.WriteString(strings.Repeat("-", 57))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", atlantisUrl))

	return handOffData.Bytes()
}

func PrintSectionMuseum() []byte {

	var chartmuseumURL string
	if viper.GetString("cloud") == pkg.CloudK3d {
		chartmuseumURL = "http://localhost:8181"
	} else {
		chartmuseumURL = fmt.Sprintf("https://chartmuseum.%s", viper.GetString("aws.hostedzonename"))
	}

	var handOffData bytes.Buffer
	handOffData.WriteString("\n--- Chartmuseum ")
	handOffData.WriteString(strings.Repeat("-", 54))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", chartmuseumURL))

	if viper.GetString("cloud") == pkg.CloudK3d {
		return handOffData.Bytes()
	} else {
		handOffData.WriteString("\n see vault for credentials ")

		return handOffData.Bytes()
	}

}

func PrintSectionMetaphor() []byte {
	var handOffData bytes.Buffer

	handOffData.WriteString("\n--- Metaphor ")
	handOffData.WriteString(strings.Repeat("-", 57))
	handOffData.WriteString(fmt.Sprintf("\n Development: %s", fmt.Sprintf("https://metaphor-development.%s", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(fmt.Sprintf("\n Staging: %s", fmt.Sprintf("https://metaphor-staging.%s", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(fmt.Sprintf("\n Production:  %s\n", fmt.Sprintf("https://metaphor-production.%s", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(strings.Repeat("-", 70))

	return handOffData.Bytes()
}
func PrintSectionMetaphorGo() []byte {
	var handOffData bytes.Buffer

	handOffData.WriteString("\n--- Metaphor Go")
	handOffData.WriteString(strings.Repeat("-", 55))
	handOffData.WriteString(fmt.Sprintf("\n Development: %s", fmt.Sprintf("https://metaphor-go-development.%s", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(fmt.Sprintf("\n Staging: %s", fmt.Sprintf("https://metaphor-go-staging.%s", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(fmt.Sprintf("\n Production:  %s\n", fmt.Sprintf("https://metaphor-go-production.%s", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(strings.Repeat("-", 70))

	return handOffData.Bytes()
}

func PrintSectionMetaphorFrontend() []byte {
	var handOffData bytes.Buffer
	if viper.GetString("cloud") == pkg.CloudK3d {
		handOffData.WriteString("\n--- Metaphor ")
		handOffData.WriteString(strings.Repeat("-", 57))
		handOffData.WriteString("\n To access the metaphor applications you'll need to \n`kubectl port-forward` to the kubernetes service")
		handOffData.WriteString("\n\n kubectl -n development port-forward svc/metaphor-frontend-development 3000:443")
		handOffData.WriteString("\n http://localhost:4000\n")
		handOffData.WriteString("\n kubectl -n staging port-forward svc/metaphor-frontend-staging 3001:443")
		handOffData.WriteString("\n http://localhost:4001\n")
		handOffData.WriteString("\n kubectl -n production port-forward svc/metaphor-frontend-production 3002:443")
		handOffData.WriteString("\n http://localhost:4002\n")
		handOffData.WriteString(strings.Repeat("-", 70))

		return handOffData.Bytes()
	} else {
		var handOffData bytes.Buffer
		handOffData.WriteString("\n--- Metaphor Frontend")
		handOffData.WriteString(strings.Repeat("-", 57))
		handOffData.WriteString(fmt.Sprintf("\n Development: %s", fmt.Sprintf("https://metaphor-frontend-development.%s", viper.GetString("aws.hostedzonename"))))
		handOffData.WriteString(fmt.Sprintf("\n Staging: %s", fmt.Sprintf("https://metaphor-frontend-staging.%s", viper.GetString("aws.hostedzonename"))))
		handOffData.WriteString(fmt.Sprintf("\n Production:  %s\n", fmt.Sprintf("https://metaphor-frontend-production.%s", viper.GetString("aws.hostedzonename"))))
		handOffData.WriteString(strings.Repeat("-", 70))

		return handOffData.Bytes()
	}
}

// HandoffScreen - prints the handoff screen
func HandoffScreen(dryRun bool, silentMode bool) {
	// prepare data for the handoff report
	if dryRun {
		log.Printf("[#99] Dry-run mode, HandoffScreen skipped.")
		return
	}

	if silentMode {
		log.Printf("[#99] Silent mode enabled, HandoffScreen skipped, please check ~/.kubefirst file for your cluster and service credentials.")
		return
	}

	var handOffData bytes.Buffer
	handOffData.Write(PrintSectionOverview())
	handOffData.Write(PrintSectionAws())
	if viper.GetString("gitprovider") == "github" {
		handOffData.Write(PrintSectionRepoGithub())
	} else {
		handOffData.Write(PrintSectionRepoGitlab())
	}
	handOffData.Write(PrintSectionVault())
	handOffData.Write(PrintSectionArgoCD())
	handOffData.Write(PrintSectionArgoWorkflows())
	handOffData.Write(PrintSectionAtlantis())
	handOffData.Write(PrintSectionMuseum())
	handOffData.Write(PrintSectionMetaphorFrontend())
	handOffData.Write(PrintSectionMetaphorGo())
	handOffData.Write(PrintSectionMetaphor())

	CommandSummary(handOffData)

}

// HandoffScreen - prints the handoff screen
func LocalHandoffScreen(dryRun bool, silentMode bool) {
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
	handOffData.Write(PrintSectionOverview())
	handOffData.Write(PrintSectionRepoGithub())
	handOffData.Write(PrintSectionVault())
	handOffData.Write(PrintSectionArgoCD())
	handOffData.Write(PrintSectionArgoWorkflows())
	handOffData.Write(PrintSectionAtlantis())
	handOffData.Write(PrintSectionMuseum())
	handOffData.Write(PrintSectionMetaphorFrontend())

	CommandSummary(handOffData)

}

func GitHubAuthToken(userCode, verificationUri string) string {
	var gitHubTokenReport bytes.Buffer
	gitHubTokenReport.WriteString(strings.Repeat("-", 69))
	gitHubTokenReport.WriteString("\nNo KUBEFIRST_GITHUB_AUTH_TOKEN env variable found!\nUse the code below to get a temporary GitHub Access Token and continue\n")
	gitHubTokenReport.WriteString(strings.Repeat("-", 69) + "\n")
	gitHubTokenReport.WriteString("1. copy the code: 📋 " + userCode + " 📋\n\n")
	gitHubTokenReport.WriteString("2. paste the code at the GitHub page: " + verificationUri + "\n")
	gitHubTokenReport.WriteString("3. authorize your organization")
	gitHubTokenReport.WriteString("\n\nA GitHub Access Token is required to provision GitHub repositories and run workflows in GitHub.")

	return gitHubTokenReport.String()
}

// LocalConnectSummary builds a string containing local service URLs
func LocalConnectSummary() string {

	var localConnect bytes.Buffer

	localConnect.WriteString(strings.Repeat("-", 70))
	localConnect.WriteString("\nKubefirst Local:\n")
	localConnect.WriteString(strings.Repeat("-", 70))

	localConnect.WriteString(fmt.Sprintf("\n\nKubefirst Console UI: %s\n", pkg.KubefirstConsoleLocalURL))
	localConnect.WriteString(fmt.Sprintf("ChartmuseumLocalURL: %s\n", pkg.ChartmuseumLocalURL))
	localConnect.WriteString(fmt.Sprintf("Argo: %s\n", pkg.ArgoLocalURL))
	localConnect.WriteString(fmt.Sprintf("ArgoCD: %s\n", pkg.ArgoCDLocalURL))
	localConnect.WriteString(fmt.Sprintf("Vault: %s\n", pkg.VaultLocalURL))
	localConnect.WriteString(fmt.Sprintf("Atlantis: %s\n", pkg.AtlantisLocalURL))
	localConnect.WriteString(fmt.Sprintf("Minio: %s\n", pkg.MinioURL))
	localConnect.WriteString(fmt.Sprintf("Minio Console: %s\n", pkg.MinioConsoleURL))

	return localConnect.String()
}
