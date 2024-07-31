/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.

Emojis definition https://github.com/yuin/goldmark-emoji/blob/master/definition/github.go
Color definition https://www.ditig.com/256-colors-cheat-sheet
*/
package progress

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/kubefirst/internal/cluster"
	"github.com/spf13/viper"
)

func renderMessage(message string) string {
	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(StyleConfig),
		glamour.WithEmoji(),
	)

	out, err := r.Render(message)
	if err != nil {
		log.Println(err.Error())
		return err.Error()
	}
	return out
}

func createStep(message string) addStep {
	out := renderMessage(message)

	return addStep{
		message: out,
	}
}

func createErrorLog(message string) errorMsg {
	out := renderMessage(fmt.Sprintf("##### :no_entry_sign: Error: %s", message))

	return errorMsg{
		message: out,
	}
}

// Public Progress Functions
func DisplayLogHints(estimatedTime int) {
	logFile := viper.GetString("k1-paths.log-file")
	cloudProvider := viper.GetString("kubefirst.cloud-provider")

	documentationLink := "https://docs.kubefirst.io/"
	if cloudProvider != "" {
		documentationLink = documentationLink + cloudProvider + `/quick-start/install/cli`
	}

	header := `
##
# Welcome to Kubefirst

### :bulb: To view verbose logs run below command in new terminal:
` + fmt.Sprintf("##### **tail -f -n +1 %s**", logFile) + `
### :blue_book: Documentation: ` + documentationLink + `

### :alarm_clock: Estimated time:` + fmt.Sprintf("`%s minutes` \n\n", strconv.Itoa(estimatedTime))

	headerMessage := renderMessage(header)

	if !CanRunBubbleTea {
		fmt.Print(headerMessage)
		return
	}

	Progress.Send(headerMsg{
		message: headerMessage,
	})
}

func DisplaySuccessMessage(cluster types.Cluster) successMsg {
	cloudCliKubeconfig := ""

	gitProviderLabel := "GitHub"

	if cluster.GitProvider == "gitlab" {
		gitProviderLabel = "GitLab"
	}

	switch cluster.CloudProvider {
	case "aws":
		cloudCliKubeconfig = fmt.Sprintf("aws eks update-kubeconfig --name %s --region %s", cluster.ClusterName, cluster.CloudRegion)
		break

	case "civo":
		cloudCliKubeconfig = fmt.Sprintf("civo kubernetes config %s --save", cluster.ClusterName)
		break

	case "digitalocean":
		cloudCliKubeconfig = "doctl kubernetes cluster kubeconfig save " + cluster.ClusterName
		break

	case "google":
		cloudCliKubeconfig = fmt.Sprintf("gcloud container clusters get-credentials %s --region=%s", cluster.ClusterName, cluster.CloudRegion)
		break

	case "vultr":
		cloudCliKubeconfig = fmt.Sprintf("vultr-cli kubernetes config %s", cluster.ClusterName)
		break

	case "k3s":
		cloudCliKubeconfig = "use the kubeconfig file outputed from terraform to acces to the cluster"
		break

	}

	var fullDomainName string

	if cluster.SubdomainName != "" {
		fullDomainName = fmt.Sprintf("%s.%s", cluster.SubdomainName, cluster.DomainName)
	} else {
		fullDomainName = cluster.DomainName
	}

	success := `
##
#### :tada: Success` + "`Cluster " + cluster.ClusterName + " is now up and running`" + `

# Cluster ` + cluster.ClusterName + `” details:

### :bulb: To retrieve root credentials for your Kubefirst platform run:
##### kubefirst ` + cluster.CloudProvider + ` root-credentials

## ` + fmt.Sprintf("`%s `", gitProviderLabel) + `
### Git Owner   ` + fmt.Sprintf("`%s`", cluster.GitAuth.Owner) + `
### Repos       ` + fmt.Sprintf("`https://%s.com/%s/gitops` \n\n", cluster.GitProvider, cluster.GitAuth.Owner) +
		fmt.Sprintf("`            https://%s.com/%s/metaphor`", cluster.GitProvider, cluster.GitAuth.Owner) + `
## Kubefirst Console
### URL         ` + fmt.Sprintf("`https://kubefirst.%s`", fullDomainName) + `
## Argo CD
### URL         ` + fmt.Sprintf("`https://argocd.%s`", fullDomainName) + `
## Vault 
### URL         ` + fmt.Sprintf("`https://vault.%s`", fullDomainName) + `


### :bulb: Quick start examples:

### To connect to your new Kubernetes cluster run:
##### ` + cloudCliKubeconfig + `

### To view all cluster pods run:
##### kubectl get pods -A
`
	successMessage := renderMessage(success)

	if !CanRunBubbleTea {
		fmt.Print(successMessage)
		return successMsg{}
	}

	return successMsg{
		message: successMessage,
	}
}

func DisplayCredentials(cluster types.Cluster) {
	header := `
##
# Root Credentials

### :bulb: Keep this data secure. These passwords can be used to access the following applications in your platform

## ArgoCD Admin Password
##### ` + cluster.ArgoCDPassword + `

## KBot User Password
##### ` + cluster.VaultAuth.KbotPassword + `

## Vault Root Token
##### ` + cluster.VaultAuth.RootToken + `
`

	headerMessage := renderMessage(header)

	if !CanRunBubbleTea {
		fmt.Print(headerMessage)
		return
	}

	Progress.Send(headerMsg{
		message: headerMessage,
	})

	Progress.Quit()
}

func AddStep(message string) {
	renderedMessage := createStep(fmt.Sprintf("%s %s", ":dizzy:", message))
	if !CanRunBubbleTea {
		fmt.Print(renderedMessage)
		return
	}

	Progress.Send(renderedMessage)
}

func CompleteStep(message string) {
	if !CanRunBubbleTea {
		fmt.Print(message)
		return
	}

	Progress.Send(completeStep{
		message: message,
	})
}

func Success(success string) {
	successMessage := renderMessage(success)

	if !CanRunBubbleTea {
		fmt.Print(successMessage)
		return
	}

	Progress.Send(
		successMsg{
			message: successMessage,
		})
}

func Error(message string) {
	renderedMessage := createErrorLog(message)

	if !CanRunBubbleTea {
		fmt.Print(renderedMessage)
		return
	}

	Progress.Send(renderedMessage)
}

func StartProvisioning(clusterName string) {
	if !CanRunBubbleTea {
		// Checks cluster status every 10 seconds
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		done := make(chan bool)

		go func() {
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					provisioningCluster, _ := cluster.GetCluster(clusterName)

					if provisioningCluster.Status == "error" {
						fmt.Printf("unable to provision cluster: %s", provisioningCluster.LastCondition)
						done <- true
					}

					if provisioningCluster.Status == "provisioned" {
						fmt.Println("\n cluster has been provisioned via ci")
						fmt.Println(fmt.Sprintf("\n kubefirst URL: https://kubefirst.%s", provisioningCluster.DomainName))
						done <- true
					}
				}
			}
		}()

		// waits until the provision is done
		<-done

	} else {
		provisioningMessage := startProvision{
			clusterName: clusterName,
		}

		Progress.Send(provisioningMessage)
	}
}
