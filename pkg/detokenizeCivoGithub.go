package pkg

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// DetokenizeCivoGithub - Translate tokens by values on a given path
func DetokenizeCivoGithub(path string) {

	err := filepath.Walk(path, DetokenizeDirectoryCivoGithub)
	if err != nil {
		log.Panic(err)
	}
}

// DetokenizeDirectory - Translate tokens by values on a directory level.
func DetokenizeDirectoryCivoGithub(path string, fi os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if fi.IsDir() {
		return nil
	}

	if strings.Contains(path, ".gitClient") || strings.Contains(path, ".terraform") || strings.Contains(path, ".git/") {
		return nil
	}

	matched, err := filepath.Match("*", fi.Name())

	if err != nil {
		log.Panic(err)
	}
	if matched {
		read, err := os.ReadFile(path)
		if err != nil {
			log.Panic(err)
		}

		// config := configs.ReadConfig()

		newContents := string(read)

		gitopsGitURL := viper.GetString("github.repo.gitops.giturl")
		civoDns := viper.GetString("civo.dns")
		atlantisWebhookURL := viper.GetString("github.atlantis.webhook.url")
		adminEmail := viper.GetString("admin-email")
		clusterName := viper.GetString("kubefirst.cluster-name")
		githubHost := viper.GetString("github.host")
		githubOwner := viper.GetString("github.owner")
		githubUser := viper.GetString("github.user")
		kubefirstStateStoreBucket := viper.GetString("kubefirst.state-store.bucket")

		//! computed
		argocdIngressURL := fmt.Sprintf("https://argocd.%s", civoDns)
		argocdIngressNoHttpsURL := fmt.Sprintf("argocd.%s", civoDns)
		argoWorkflowsIngressURL := fmt.Sprintf("https://argo.%s", civoDns)
		argoWorkflowsIngressNoHttpsURL := fmt.Sprintf("argo.%s", civoDns)
		gitopsNoHttpsURL := fmt.Sprintf("github.com/%s/gitops.git", viper.GetString("github.owner"))
		gitopsURL := fmt.Sprintf("https://github.com/%s/gitops.git", viper.GetString("github.owner"))
		vaultIngressURL := fmt.Sprintf("https://vault.%s", civoDns)
		vaultIngressNoHttpsURL := fmt.Sprintf("vault.%s", civoDns)
		vouchIngressURL := fmt.Sprintf("https://vouch.%s", civoDns)
		// kubefirstIngressURL := fmt.Sprintf("kubefirst.%s", civoDns)
		atlantisIngressNoHttpsURL := fmt.Sprintf("atlantis.%s", civoDns)
		atlantisIngressURL := fmt.Sprintf("https://atlantis.%s", civoDns)

		// todo consolidate
		metaphorDevelopmentIngressNoHttpsURL := fmt.Sprintf("metaphor-development.%s", civoDns)
		metaphorStagingIngressNoHttpsURL := fmt.Sprintf("metaphor-staging.%s", civoDns)
		metaphorProductionIngressNoHttpsURL := fmt.Sprintf("metaphor-production.%s", civoDns)
		metaphorDevelopmentIngressURL := fmt.Sprintf("https://metaphor-development.%s", civoDns)
		metaphorStagingIngressURL := fmt.Sprintf("https://metaphor-staging.%s", civoDns)
		metaphorProductionIngressURL := fmt.Sprintf("https://metaphor-production.%s", civoDns)
		// todo consolidate
		metaphorFrontendDevelopmentIngressNoHttpsURL := fmt.Sprintf("metaphor-frontend-development.%s", civoDns)
		metaphorFrontendStagingIngressNoHttpsURL := fmt.Sprintf("metaphor-frontend-staging.%s", civoDns)
		metaphorFrontendProductionIngressNoHttpsURL := fmt.Sprintf("metaphor-frontend-production.%s", civoDns)
		metaphorFrontendDevelopmentIngressURL := fmt.Sprintf("https://metaphor-frontend-development.%s", civoDns)
		metaphorFrontendStagingIngressURL := fmt.Sprintf("https://metaphor-frontend-staging.%s", civoDns)
		metaphorFrontendProductionIngressURL := fmt.Sprintf("https://metaphor-frontend-production.%s", civoDns)
		// todo consolidate
		metaphorGoDevelopmentIngressNoHttpsURL := fmt.Sprintf("metaphor-go-development.%s", civoDns)
		metaphorGoStagingIngressNoHttpsURL := fmt.Sprintf("metaphor-go-staging.%s", civoDns)
		metaphorGoProductionIngressNoHttpsURL := fmt.Sprintf("metaphor-go-production.%s", civoDns)
		metaphorGoDevelopmentIngressURL := fmt.Sprintf("https://metaphor-go-development.%s", civoDns)
		metaphorGoStagingIngressURL := fmt.Sprintf("https://metaphor-go-staging.%s", civoDns)
		metaphorGoProductionIngressURL := fmt.Sprintf("https://metaphor-go-production.%s", civoDns)

		newContents = strings.Replace(newContents, "<GIT_PROVIDER>", "GitHub", -1)
		newContents = strings.Replace(newContents, "<GIT_NAMESPACE>", "N/A", -1)
		newContents = strings.Replace(newContents, "<GIT_DESCRIPTION>", "GitHub hosted git", -1)
		newContents = strings.Replace(newContents, "<GIT_URL>", gitopsURL, -1)
		newContents = strings.Replace(newContents, "<GIT_RUNNER>", "GitHub Action Runner", -1)
		newContents = strings.Replace(newContents, "<GIT_RUNNER_NS>", "github-runner", -1)
		newContents = strings.Replace(newContents, "<GIT_RUNNER_DESCRIPTION>", "Self Hosted GitHub Action Runner", -1)

		newContents = strings.Replace(newContents, "<ADMIN_EMAIL_ADDRESS>", adminEmail, -1)
		newContents = strings.Replace(newContents, "<ARGO_CD_INGRESS_URL>", argocdIngressURL, -1)
		newContents = strings.Replace(newContents, "<ARGO_WORKFLOWS_INGRESS_URL>", argoWorkflowsIngressURL, -1)
		newContents = strings.Replace(newContents, "<ATLANTIS_INGRESS_URL>", atlantisIngressURL, -1)
		newContents = strings.Replace(newContents, "<CIVO_DNS>", civoDns, -1)
		newContents = strings.Replace(newContents, "<CLUSTER_NAME>", clusterName, -1)
		//! registry
		newContents = strings.Replace(newContents, "<GITHUB_HOST>", githubHost, -1)
		newContents = strings.Replace(newContents, "<GITHUB_OWNER>", githubOwner, -1)
		newContents = strings.Replace(newContents, "<GITHUB_USER>", githubUser, -1)
		newContents = strings.Replace(newContents, "<GITOPS_REPO_ATLANTIS_WEBHOOK_URL>", atlantisWebhookURL, -1)
		newContents = strings.Replace(newContents, "<GITOPS_REPO_GIT_URL>", gitopsGitURL, -1)
		newContents = strings.Replace(newContents, "<GITOPS_REPO_NO_HTTPS_URL>", gitopsNoHttpsURL, -1)
		newContents = strings.Replace(newContents, "<KUBEFIRST_VERSION>", "0.0.0", -1) // TODO NEED TO REVIEW THIS
		// todo need METAPHOR_*_INGRESS_NO_HTTPS_URL variations for hosts on ingress resources
		newContents = strings.Replace(newContents, "<METAPHOR_DEVELPOMENT_INGRESS_URL>", metaphorDevelopmentIngressURL, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_STAGING_INGRESS_URL>", metaphorStagingIngressURL, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_PRODUCTION_INGRESS_URL>", metaphorProductionIngressURL, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_FRONT_DEVELOPMENT_INGRESS_URL>", metaphorFrontendDevelopmentIngressURL, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_FRONT_STAGING_INGRESS_URL>", metaphorFrontendStagingIngressURL, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_FRONT_PRODUCTION_INGRESS_URL>", metaphorFrontendProductionIngressURL, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_GO_DEVELOPMENT_INGRESS_URL>", metaphorGoDevelopmentIngressURL, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_GO_STAGING_INGRESS_URL>", metaphorGoStagingIngressURL, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_GO_PRODUCTION_INGRESS_URL>", metaphorGoProductionIngressURL, -1)
		newContents = strings.Replace(newContents, "<VAULT_INGRESS_NO_HTTPS_URL>", vaultIngressNoHttpsURL, -1)
		newContents = strings.Replace(newContents, "<VAULT_INGRESS_URL>", vaultIngressURL, -1)
		newContents = strings.Replace(newContents, "<VOUCH_INGRESS_URL>", vouchIngressURL, -1)

		// todo consolidate this?
		newContents = strings.Replace(newContents, "<METAPHOR_DEVELOPMENT_INGRESS_NO_HTTPS_URL>", metaphorDevelopmentIngressNoHttpsURL, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_STAGING_INGRESS_NO_HTTPS_URL>", metaphorStagingIngressNoHttpsURL, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_PRODUCTION_INGRESS_NO_HTTPS_URL>", metaphorProductionIngressNoHttpsURL, -1)

		newContents = strings.Replace(newContents, "<METAPHOR_FRONTEND_DEVELOPMENT_INGRESS_NO_HTTPS_URL>", metaphorFrontendDevelopmentIngressNoHttpsURL, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_FRONTEND_STAGING_INGRESS_NO_HTTPS_URL>", metaphorFrontendStagingIngressNoHttpsURL, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_FRONTEND_PRODUCTION_INGRESS_NO_HTTPS_URL>", metaphorFrontendProductionIngressNoHttpsURL, -1)

		newContents = strings.Replace(newContents, "<METAPHOR_GO_DEVELOPMENT_INGRESS_NO_HTTPS_URL>", metaphorGoDevelopmentIngressNoHttpsURL, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_GO_STAGING_INGRESS_NO_HTTPS_URL>", metaphorGoStagingIngressNoHttpsURL, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_GO_PRODUCTION_INGRESS_NO_HTTPS_URL>", metaphorGoProductionIngressNoHttpsURL, -1)

		newContents = strings.Replace(newContents, "<KUBEFIRST_VERSION>", "TODO", -1) // todo get version

		//! terraform
		// ? argocd ingress url might be in registry?
		newContents = strings.Replace(newContents, "<ARGOCD_INGRESS_URL>", argocdIngressURL, -1)
		newContents = strings.Replace(newContents, "<ARGOCD_INGRESS_NO_HTTP_URL>", argocdIngressNoHttpsURL, -1)

		// didn't see the below tokens
		newContents = strings.Replace(newContents, "<ARGO_WORKFLOWS_INGRESS_NO_HTTPS_URL>", argoWorkflowsIngressNoHttpsURL, -1)
		newContents = strings.Replace(newContents, "<ATLANTIS_INGRESS_NO_HTTPS_URL>", atlantisIngressNoHttpsURL, -1)
		newContents = strings.Replace(newContents, "<KUBEFIRST_STATE_STORE_BUCKET>", kubefirstStateStoreBucket, -1)

		err = os.WriteFile(path, []byte(newContents), 0)
		if err != nil {
			log.Panic(err)
		}
	}

	return nil
}
