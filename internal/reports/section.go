package reports

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/kubefirst/kubefirst/internal/flagset"
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
	handOffData.WriteString(fmt.Sprintf("\nSave this information for future use, once you leave this screen some of this information is lost. "))
	handOffData.WriteString(fmt.Sprintf("\n\nAccess the Console on your Browser at: http://localhost:9094\n"))
	handOffData.WriteString(fmt.Sprintf("\nPress ESC to leave this screen and return to shell."))

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
	if viper.GetString("cloud") == flagset.CloudK3d {
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
	if viper.GetString("cloud") == flagset.CloudK3d {
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
	if viper.GetString("cloud") == flagset.CloudK3d {
		argoWorkflowsURL = "http://localhost:8080"
	} else {
		argoWorkflowsURL = fmt.Sprintf("https://argo.%s", viper.GetString("aws.hostedzonename"))
	}

	var handOffData bytes.Buffer
	handOffData.WriteString("\n--- Argo Workflows ")
	handOffData.WriteString(strings.Repeat("-", 51))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", argoWorkflowsURL))
	handOffData.WriteString("\n sso credentials only ")
	handOffData.WriteString("\n * sso enabled ")

	return handOffData.Bytes()
}

func PrintSectionAtlantis() []byte {
	var handOffData bytes.Buffer

	handOffData.WriteString("\n--- Atlantis ")
	handOffData.WriteString(strings.Repeat("-", 57))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", fmt.Sprintf("https://atlantis.%s", viper.GetString("aws.hostedzonename"))))

	return handOffData.Bytes()
}

func PrintSectionMuseum() []byte {
	var handOffData bytes.Buffer

	handOffData.WriteString("\n--- Museum ")
	handOffData.WriteString(strings.Repeat("-", 59))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s\n", fmt.Sprintf("https://chartmuseum.%s", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(" see vault for credentials ")

	return handOffData.Bytes()
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

	handOffData.WriteString("\n--- Metaphor Frontend")
	handOffData.WriteString(strings.Repeat("-", 49))
	handOffData.WriteString(fmt.Sprintf("\n Development: %s", fmt.Sprintf("https://metaphor-frontend-development.%s", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(fmt.Sprintf("\n Staging: %s", fmt.Sprintf("https://metaphor-frontend-staging.%s", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(fmt.Sprintf("\n Production:  %s\n", fmt.Sprintf("https://metaphor-frontend-production.%s", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(strings.Repeat("-", 70))

	return handOffData.Bytes()
}

//HandoffScreen - prints the handoff screen
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
	if viper.GetBool("github.enabled") {
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

//HandoffScreen - prints the handoff screen
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
	if viper.GetBool("github.enabled") {
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

func GitHubAuthToken(userCode, verificationUri string) string {
	var gitHubTokenReport bytes.Buffer
	gitHubTokenReport.WriteString(strings.Repeat("-", 69))
	gitHubTokenReport.WriteString("\nNo GITHUB_AUTH_TOKEN env variable found!\nUse the code below to get a temporary GitHub Personal Access Token and continue\n")
	gitHubTokenReport.WriteString(strings.Repeat("-", 69) + "\n")
	gitHubTokenReport.WriteString("1. copy the code: 📋 " + userCode + " 📋\n\n")
	gitHubTokenReport.WriteString("2. paste the code at the GitHub page: " + verificationUri + "\n")
	gitHubTokenReport.WriteString("3. authorize your organization")
	gitHubTokenReport.WriteString("\n\nA GitHub Personal Access Token is required to provision GitHub repositories and run workflows in GitHub.\n\n")

	return gitHubTokenReport.String()
}
