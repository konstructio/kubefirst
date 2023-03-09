package reports

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"

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
	if viper.GetString("cloud") == pkg.CloudK3d {
		handOffData.WriteString(fmt.Sprintf("\n  %s", fmt.Sprintf("https://%s/%s/metaphor", viper.GetString("github.host"), viper.GetString("github.owner"))))
	} else {
		handOffData.WriteString(fmt.Sprintf("\n  %s", fmt.Sprintf("https://%s/%s/metaphor", viper.GetString("github.host"), viper.GetString("github.owner"))))
		handOffData.WriteString(fmt.Sprintf("\n  %s", fmt.Sprintf("https://%s/%s/metaphor-go", viper.GetString("github.host"), viper.GetString("github.owner"))))
		handOffData.WriteString(fmt.Sprintf("\n  %s", fmt.Sprintf("https://%s/%s/metaphor", viper.GetString("github.host"), viper.GetString("github.owner"))))

	}

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
	handOffData.WriteString(fmt.Sprintf("\n  %s", fmt.Sprintf("https://gitlab.%s/kubefirst/metaphor-go", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(fmt.Sprintf("\n  %s", fmt.Sprintf("https://gitlab.%s/kubefirst/metaphor", viper.GetString("aws.hostedzonename"))))

	return handOffData.Bytes()
}

func PrintSectionOverview() []byte {
	var handOffData bytes.Buffer
	config := configs.ReadConfig()
	handOffData.WriteString(strings.Repeat("-", 70))
	handOffData.WriteString(fmt.Sprintf("\nCluster %q is up and running!:", viper.GetString("cluster-name")))
	handOffData.WriteString("\nThis information is available at $HOME/.kubefirst ")
	handOffData.WriteString("\n")
	handOffData.WriteString("\nPress ESC to leave this screen and return to your shell.")

	if viper.GetString("cloud") == pkg.CloudK3d {
		handOffData.WriteString("\n\nNotes:")
		handOffData.WriteString("\n  Kubefirst generated certificates to ensure secure connections to")
		handOffData.WriteString("\n  your local kubernetes services. However they will not")
		handOffData.WriteString("\n  trusted by your browser by default. ")
		handOffData.WriteString("\n")
		handOffData.WriteString("\n  It is safe to ignore the warning and continue to these sites. ")
		handOffData.WriteString("\n  To remove these warnings, you can install your new certificate ")
		handOffData.WriteString("\n  to your local trust store by running the following command: ")
		handOffData.WriteString(fmt.Sprintf("\n    %s -install", config.MkCertPath))
		handOffData.WriteString("\n  For more details on the mkcert utility, please see:")
		handOffData.WriteString("\n  https://github.com/FiloSottile/mkcert#changing-the-location-of-the-ca-files")
	}
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
		vaultURL = pkg.VaultLocalURLTLS
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
		argoCdURL = pkg.ArgoCDLocalURLTLS
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
		argoWorkflowsURL = pkg.ArgoLocalURLTLS
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

	var atlantisURL string
	if viper.GetString("cloud") == pkg.CloudK3d {
		atlantisURL = pkg.AtlantisLocalURLTLS
	} else {
		atlantisURL = fmt.Sprintf("https://atlantis.%s", viper.GetString("aws.hostedzonename"))
	}

	var handOffData bytes.Buffer
	handOffData.WriteString("\n--- Atlantis ")
	handOffData.WriteString(strings.Repeat("-", 57))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", atlantisURL))

	return handOffData.Bytes()
}

func PrintSectionMuseum() []byte {

	var chartmuseumURL string
	if viper.GetString("cloud") == pkg.CloudK3d {
		chartmuseumURL = pkg.ChartmuseumLocalURLTLS
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
	handOffData.WriteString(fmt.Sprintf("\n Development: %s", fmt.Sprintf("https://metaphor-development.%s/app", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(fmt.Sprintf("\n Staging: %s", fmt.Sprintf("https://metaphor-staging.%s/app", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(fmt.Sprintf("\n Production:  %s\n", fmt.Sprintf("https://metaphor-production.%s/app", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(strings.Repeat("-", 70))

	return handOffData.Bytes()
}
func PrintSectionMetaphorGo() []byte {
	var handOffData bytes.Buffer

	handOffData.WriteString("\n--- Metaphor Go")
	handOffData.WriteString(strings.Repeat("-", 55))
	handOffData.WriteString(fmt.Sprintf("\n Development: %s", fmt.Sprintf("https://metaphor-go-development.%s/app", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(fmt.Sprintf("\n Staging: %s", fmt.Sprintf("https://metaphor-go-staging.%s/app", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(fmt.Sprintf("\n Production:  %s\n", fmt.Sprintf("https://metaphor-go-production.%s/app", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(strings.Repeat("-", 70))

	return handOffData.Bytes()
}

func PrintSectionMetaphorFrontend() []byte {

	var handOffData bytes.Buffer

	if viper.GetString("cloud") == pkg.CloudK3d {
		handOffData.WriteString("\n--- Metaphor Slim ")
		handOffData.WriteString(strings.Repeat("-", 52))
		handOffData.WriteString(fmt.Sprintf("\n URL: %s\n", pkg.MetaphorFrontendSlimTLSDev))
		handOffData.WriteString(strings.Repeat("-", 70))

		return handOffData.Bytes()
	}

	handOffData.WriteString("\n--- Metaphor Frontend")
	handOffData.WriteString(strings.Repeat("-", 57))
	handOffData.WriteString(fmt.Sprintf("\n Development: %s", fmt.Sprintf("https://metaphor-development.%s", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(fmt.Sprintf("\n Staging: %s", fmt.Sprintf("https://metaphor-staging.%s", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(fmt.Sprintf("\n Production:  %s\n", fmt.Sprintf("https://metaphor-production.%s", viper.GetString("aws.hostedzonename"))))
	handOffData.WriteString(strings.Repeat("-", 70))

	return handOffData.Bytes()
}

func PrintSectionConsole(consoleURL string) []byte {

	var handOffData bytes.Buffer
	handOffData.WriteString("\n--- Kubefirst Console ")
	handOffData.WriteString(strings.Repeat("-", 48))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", consoleURL))

	return handOffData.Bytes()
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
	if viper.GetString("git-provider") == "github" {
		handOffData.Write(PrintSectionRepoGithub())
	} else {
		handOffData.Write(PrintSectionRepoGitlab())
	}
	handOffData.Write(PrintSectionConsole(pkg.KubefirstConsoleLocalURLCloud))
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

// LocalHandoffScreen prints the handoff screen
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
	handOffData.Write(PrintSectionConsole(pkg.KubefirstConsoleLocalURLTLS))
	handOffData.Write(PrintSectionVault())
	handOffData.Write(PrintSectionArgoCD())
	handOffData.Write(PrintSectionArgoWorkflows())
	handOffData.Write(PrintSectionAtlantis())
	handOffData.Write(PrintSectionMuseum())
	handOffData.Write(PrintSectionMetaphorFrontend())

	CommandSummary(handOffData)

}

// CivoHandoff prints the handoff screen
func CivoHandoff(clusterName string, domainName string) {

	var handOffData bytes.Buffer
	handOffData.WriteString(strings.Repeat("-", 70))
	handOffData.WriteString(fmt.Sprintf("\nCluster %q is up and running!", clusterName))
	handOffData.WriteString("\n\nIf you close this window you can find these values in")
	handOffData.WriteString("\nThe platform details are available at `$HOME/.kubefirst`")

	handOffData.WriteString("\n")
	handOffData.WriteString("\n--- Vault " + strings.Repeat("-", 60))
	handOffData.WriteString(fmt.Sprintf("\n URL: https://vault.%s", domainName))
	handOffData.WriteString("\nTo access vault")
	handOffData.WriteString("\nPress ESC to leave this screen and return to your shell.")

	CommandSummary(handOffData)

}

func GitHubAuthToken(userCode, verificationUri string) string {
	var gitHubTokenReport bytes.Buffer
	gitHubTokenReport.WriteString(strings.Repeat("-", 69))
	gitHubTokenReport.WriteString("\nNo GITHUB_TOKEN env variable found!\nUse the code below to get a temporary GitHub Access Token\nThis token will be used by Kubefirst to create your environment\n")
	gitHubTokenReport.WriteString("\n\nA GitHub Access Token is required to provision GitHub repositories and run workflows in GitHub.\n")
	gitHubTokenReport.WriteString(strings.Repeat("-", 69) + "\n")
	gitHubTokenReport.WriteString("1. Copy this code: 📋 " + userCode + " 📋\n\n")
	gitHubTokenReport.WriteString(fmt.Sprintf("2. When ready, press <enter> to open the page at %s\n\n", verificationUri))
	gitHubTokenReport.WriteString("3. Authorize the organization you'll be using Kubefirst with - this may also be your personal account")

	return gitHubTokenReport.String()
}

// LocalConnectSummary builds a string containing local service URLs
func LocalConnectSummary() string {

	config := configs.ReadConfig()

	var localConnect bytes.Buffer

	localConnect.WriteString(strings.Repeat("-", 70))
	localConnect.WriteString("\nKubefirst Local:\n")
	localConnect.WriteString(strings.Repeat("-", 70))

	localConnect.WriteString(fmt.Sprintf("\n\nKubefirst Console:    %s\n", pkg.KubefirstConsoleLocalURL))
	localConnect.WriteString(fmt.Sprintf("Chart Museum:         %s\n", pkg.ChartmuseumLocalURL))
	localConnect.WriteString(fmt.Sprintf("Argo:                 %s/workflows\n", config.ArgoWorkflowsLocalURL))
	localConnect.WriteString(fmt.Sprintf("ArgoCD:               %s\n", pkg.ArgoCDLocalURL))
	localConnect.WriteString(fmt.Sprintf("Vault:                %s\n", pkg.VaultLocalURL))
	localConnect.WriteString(fmt.Sprintf("Atlantis:             %s\n", pkg.AtlantisLocalURL))
	localConnect.WriteString(fmt.Sprintf("Minio Console:        %s\n", "pkg.MinioConsoleURL")) // todo figure out the source of truth and fix
	localConnect.WriteString(fmt.Sprintf("Metaphor Frontend:    %s\n", pkg.MetaphorFrontendDevelopmentLocalURL))
	localConnect.WriteString(fmt.Sprintf("Metaphor Go:          %s/app\n", pkg.MetaphorGoDevelopmentLocalURL))
	localConnect.WriteString(fmt.Sprintf("Metaphor:             %s/app\n", pkg.MetaphorDevelopmentLocalURL))

	return localConnect.String()
}
