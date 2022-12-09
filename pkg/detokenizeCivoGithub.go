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

		gitopsGitUrl := viper.GetString("github.repo.gitops.giturl")
		civoDns := viper.GetString("civo.dns")
		atlantisWebhookUrl := viper.GetString("github.atlantis.webhook.url")
		adminEmail := viper.GetString("admin-email")
		clusterName := viper.GetString("cluster-name")
		githubHost := viper.GetString("github.host")
		githubOwner := viper.GetString("github.owner")
		githubUser := viper.GetString("github.user")
		kubefirstStateStoreBucket := viper.GetString("kubefirst.state-store.bucket")

		//! computed
		argocdIngressUrl := fmt.Sprintf("https://argocd.%s", civoDns)
		argocdIngressNoHttpsUrl := fmt.Sprintf("argocd.%s", civoDns)
		argoWorkflowsIngressUrl := fmt.Sprintf("https://argo.%s", civoDns)
		argoWorkflowsIngressNoHttpsUrl := fmt.Sprintf("argo.%s", civoDns)
		gitopsNoHttpsUrl := fmt.Sprintf("github.com/%s/gitops.git", viper.GetString("github.owner"))
		gitopsUrl := fmt.Sprintf("https://github.com/%s/gitops.git", viper.GetString("github.owner"))
		vaultIngressUrl := fmt.Sprintf("https://vault.%s", civoDns)
		vaultIngressNoHttpsUrl := fmt.Sprintf("vault.%s", civoDns)
		vouchIngressUrl := fmt.Sprintf("https://vouch.%s", civoDns)
		// kubefirstIngressUrl := fmt.Sprintf("kubefirst.%s", civoDns)
		atlantisIngressNoHttpsUrl := fmt.Sprintf("atlantis.%s", civoDns)
		atlantisIngressUrl := fmt.Sprintf("https://atlantis.%s", civoDns)
		gitlabIngressUrl := fmt.Sprintf("https://gitlab.%s", civoDns)

		// todo consolidate
		metaphorDevelopmentIngressNoHttpsUrl := fmt.Sprintf("metaphor-development.%s", civoDns)
		metaphorStagingIngressNoHttpsUrl := fmt.Sprintf("metaphor-staging.%s", civoDns)
		metaphorProductionIngressNoHttpsUrl := fmt.Sprintf("metaphor-production.%s", civoDns)
		metaphorDevelopmentIngressUrl := fmt.Sprintf("https://metaphor-development.%s", civoDns)
		metaphorStagingIngressUrl := fmt.Sprintf("https://metaphor-staging.%s", civoDns)
		metaphorProductionIngressUrl := fmt.Sprintf("https://metaphor-production.%s", civoDns)
		// todo consolidate
		metaphorFrontendDevelopmentIngressNoHttpsUrl := fmt.Sprintf("metaphor-frontend-development.%s", civoDns)
		metaphorFrontendStagingIngressNoHttpsUrl := fmt.Sprintf("metaphor-frontend-staging.%s", civoDns)
		metaphorFrontendProductionIngressNoHttpsUrl := fmt.Sprintf("metaphor-frontend-production.%s", civoDns)
		metaphorFrontendDevelopmentIngressUrl := fmt.Sprintf("https://metaphor-frontend-development.%s", civoDns)
		metaphorFrontendStagingIngressUrl := fmt.Sprintf("https://metaphor-frontend-staging.%s", civoDns)
		metaphorFrontendProductionIngressUrl := fmt.Sprintf("https://metaphor-frontend-production.%s", civoDns)
		// todo consolidate
		metaphorGoDevelopmentIngressNoHttpsUrl := fmt.Sprintf("metaphor-go-development.%s", civoDns)
		metaphorGoStagingIngressNoHttpsUrl := fmt.Sprintf("metaphor-go-staging.%s", civoDns)
		metaphorGoProductionIngressNoHttpsUrl := fmt.Sprintf("metaphor-go-production.%s", civoDns)
		metaphorGoDevelopmentIngressUrl := fmt.Sprintf("https://metaphor-go-development.%s", civoDns)
		metaphorGoStagingIngressUrl := fmt.Sprintf("https://metaphor-go-staging.%s", civoDns)
		metaphorGoProductionIngressUrl := fmt.Sprintf("https://metaphor-go-production.%s", civoDns)

		newContents = strings.Replace(newContents, "<GIT_PROVIDER>", "GitHub", -1)
		newContents = strings.Replace(newContents, "<GIT_NAMESPACE>", "N/A", -1)
		newContents = strings.Replace(newContents, "<GIT_DESCRIPTION>", "GitHub hosted git", -1)
		newContents = strings.Replace(newContents, "<GIT_URL>", gitopsUrl, -1)
		newContents = strings.Replace(newContents, "<GIT_RUNNER>", "GitHub Action Runner", -1)
		newContents = strings.Replace(newContents, "<GIT_RUNNER_NS>", "github-runner", -1)
		newContents = strings.Replace(newContents, "<GIT_RUNNER_DESCRIPTION>", "Self Hosted GitHub Action Runner", -1)

		newContents = strings.Replace(newContents, "<ADMIN_EMAIL_ADDRESS>", adminEmail, -1)
		newContents = strings.Replace(newContents, "<ARGO_CD_INGRESS_URL>", argocdIngressUrl, -1)
		newContents = strings.Replace(newContents, "<ARGO_WORKFLOWS_INGRESS_URL>", argoWorkflowsIngressUrl, -1)
		newContents = strings.Replace(newContents, "<ATLANTIS_INGRESS_URL>", atlantisIngressUrl, -1)
		newContents = strings.Replace(newContents, "<CIVO_DNS>", civoDns, -1)
		newContents = strings.Replace(newContents, "<CLUSTER_NAME>", clusterName, -1)
		//! registry
		newContents = strings.Replace(newContents, "<GITHUB_HOST>", githubHost, -1)
		newContents = strings.Replace(newContents, "<GITHUB_OWNER>", githubOwner, -1)
		newContents = strings.Replace(newContents, "<GITHUB_USER>", githubUser, -1)
		newContents = strings.Replace(newContents, "<GITOPS_REPO_ATLANTIS_WEBHOOK_URL>", atlantisWebhookUrl, -1)
		newContents = strings.Replace(newContents, "<GITOPS_REPO_GIT_URL>", gitopsGitUrl, -1)
		newContents = strings.Replace(newContents, "<GITOPS_REPO_NO_HTTPS_URL>", gitopsNoHttpsUrl, -1)
		newContents = strings.Replace(newContents, "<KUBEFIRST_VERSION>", "0.0.0", -1) // TODO NEED TO REVIEW THIS
		// todo need METAPHOR_*_INGRESS_NO_HTTPS_URL variations for hosts on ingress resources
		newContents = strings.Replace(newContents, "<METAPHOR_DEVELPOMENT_INGRESS_URL>", metaphorDevelopmentIngressUrl, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_STAGING_INGRESS_URL>", metaphorStagingIngressUrl, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_PRODUCTION_INGRESS_URL>", metaphorProductionIngressUrl, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_FRONT_DEVELOPMENT_INGRESS_URL>", metaphorFrontendDevelopmentIngressUrl, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_FRONT_STAGING_INGRESS_URL>", metaphorFrontendStagingIngressUrl, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_FRONT_PRODUCTION_INGRESS_URL>", metaphorFrontendProductionIngressUrl, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_GO_DEVELOPMENT_INGRESS_URL>", metaphorGoDevelopmentIngressUrl, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_GO_STAGING_INGRESS_URL>", metaphorGoStagingIngressUrl, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_GO_PRODUCTION_INGRESS_URL>", metaphorGoProductionIngressUrl, -1)
		newContents = strings.Replace(newContents, "<VAULT_INGRESS_NO_HTTPS_URL>", vaultIngressNoHttpsUrl, -1)
		newContents = strings.Replace(newContents, "<VAULT_INGRESS_URL>", vaultIngressUrl, -1)
		newContents = strings.Replace(newContents, "<VOUCH_INGRESS_URL>", vouchIngressUrl, -1)

		if viper.GetString("terraform.aws.outputs.kms-key.id") != "" {
			awsVaultKmsKeyId := viper.GetString("terraform.aws.outputs.kms-key.id")
			newContents = strings.Replace(newContents, "<AWS_VAULT_KMS_KEY_ID>", awsVaultKmsKeyId, -1)
		}

		// todo consolidate this?
		newContents = strings.Replace(newContents, "<METAPHOR_DEVELOPMENT_INGRESS_NO_HTTPS_URL>", metaphorDevelopmentIngressNoHttpsUrl, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_STAGING_INGRESS_NO_HTTPS_URL>", metaphorStagingIngressNoHttpsUrl, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_PRODUCTION_INGRESS_NO_HTTPS_URL>", metaphorProductionIngressNoHttpsUrl, -1)

		newContents = strings.Replace(newContents, "<METAPHOR_FRONTEND_DEVELOPMENT_INGRESS_NO_HTTPS_URL>", metaphorFrontendDevelopmentIngressNoHttpsUrl, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_FRONTEND_STAGING_INGRESS_NO_HTTPS_URL>", metaphorFrontendStagingIngressNoHttpsUrl, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_FRONTEND_PRODUCTION_INGRESS_NO_HTTPS_URL>", metaphorFrontendProductionIngressNoHttpsUrl, -1)

		newContents = strings.Replace(newContents, "<METAPHOR_GO_DEVELOPMENT_INGRESS_NO_HTTPS_URL>", metaphorGoDevelopmentIngressNoHttpsUrl, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_GO_STAGING_INGRESS_NO_HTTPS_URL>", metaphorGoStagingIngressNoHttpsUrl, -1)
		newContents = strings.Replace(newContents, "<METAPHOR_GO_PRODUCTION_INGRESS_NO_HTTPS_URL>", metaphorGoProductionIngressNoHttpsUrl, -1)

		newContents = strings.Replace(newContents, "<KUBEFIRST_VERSION>", "TODO", -1) // todo get version

		//! terraform
		// ? argocd ingress url might be in registry?
		newContents = strings.Replace(newContents, "<ARGOCD_INGRESS_URL>", argocdIngressUrl, -1)
		newContents = strings.Replace(newContents, "<ARGOCD_INGRESS_NO_HTTP_URL>", argocdIngressNoHttpsUrl, -1)

		// didnt see
		newContents = strings.Replace(newContents, "<ARGO_WORKFLOWS_INGRESS_URL>", argoWorkflowsIngressUrl, -1)
		newContents = strings.Replace(newContents, "<ARGO_WORKFLOWS_INGRESS_NO_HTTPS_URL>", argoWorkflowsIngressNoHttpsUrl, -1)
		newContents = strings.Replace(newContents, "<VAULT_INGRESS_URL>", vaultIngressUrl, -1)
		newContents = strings.Replace(newContents, "<VOUCH_INGRESS_URL>", vouchIngressUrl, -1)
		newContents = strings.Replace(newContents, "<GITOPS_REPO_ATLANTIS_WEBHOOK_URL>", atlantisWebhookUrl, -1)
		newContents = strings.Replace(newContents, "<ATLANTIS_INGRESS_NO_HTTPS_URL>", atlantisIngressNoHttpsUrl, -1)
		newContents = strings.Replace(newContents, "<ATLANTIS_INGRESS_URL>", atlantisIngressUrl, -1)
		newContents = strings.Replace(newContents, "<GITLAB_INGRESS_URL>", gitlabIngressUrl, -1)
		newContents = strings.Replace(newContents, "<KUBEFIRST_STATE_STORE_BUCKET>", kubefirstStateStoreBucket, -1)

		err = os.WriteFile(path, []byte(newContents), 0)
		if err != nil {
			log.Panic(err)
		}
	}

	return nil
}
