package aws

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kubefirst/kubefirst/configs"
)

// detokenizeDirectoryRecursively - Translate tokens by values on a given path
func DetokenizeGitGitops(path string, tokens *GitOpsDirectoryValues) error {
	err := filepath.Walk(path, detokenizeGitops(path, tokens))
	if err != nil {
		return err
	}

	return nil
}

func detokenizeGitops(path string, tokens *GitOpsDirectoryValues) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !!fi.IsDir() {
			return nil
		}

		// Local values
		// todo consolidate
		metaphorDevelopmentIngressNoHttpsURL := fmt.Sprintf("metaphor-development.%s", tokens.DomainName)
		metaphorStagingIngressNoHttpsURL := fmt.Sprintf("metaphor-staging.%s", tokens.DomainName)
		metaphorProductionIngressNoHttpsURL := fmt.Sprintf("metaphor-production.%s", tokens.DomainName)
		metaphorDevelopmentIngressURL := fmt.Sprintf("https://metaphor-development.%s", tokens.DomainName)
		metaphorStagingIngressURL := fmt.Sprintf("https://metaphor-staging.%s", tokens.DomainName)
		metaphorProductionIngressURL := fmt.Sprintf("https://metaphor-production.%s", tokens.DomainName)

		// var matched bool
		matched, err := filepath.Match("*", fi.Name())
		if matched {
			read, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			newContents := string(read)
			// aws specific tokens
			newContents = strings.Replace(newContents, "<AWS_IAM_ARN_ACCOUNT_ROOT>", tokens.AwsIamArnAccountRoot, -1)
			newContents = strings.Replace(newContents, "<AWS_NODE_CAPACITY_TYPE>", tokens.AwsNodeCapacityType, -1)
			newContents = strings.Replace(newContents, "<KUBEFIRST_ARTIFACTS_BUCKET>", tokens.KubefirstArtifactsBucket, -1)
			// end aws specific
			newContents = strings.Replace(newContents, "<ALERTS_EMAIL>", tokens.AlertsEmail, -1)
			newContents = strings.Replace(newContents, "<ATLANTIS_ALLOW_LIST>", tokens.AtlantisAllowList, -1)
			newContents = strings.Replace(newContents, "<CLUSTER_NAME>", tokens.ClusterName, -1)
			newContents = strings.Replace(newContents, "<CLOUD_PROVIDER>", tokens.CloudProvider, -1)
			newContents = strings.Replace(newContents, "<CLOUD_REGION>", tokens.CloudRegion, -1)
			newContents = strings.Replace(newContents, "<CLUSTER_NAME>", tokens.ClusterName, -1)
			newContents = strings.Replace(newContents, "<CLUSTER_TYPE>", tokens.ClusterType, -1)
			newContents = strings.Replace(newContents, "<DOMAIN_NAME>", tokens.DomainName, -1)
			newContents = strings.Replace(newContents, "<KUBE_CONFIG_PATH>", tokens.Kubeconfig, -1)
			newContents = strings.Replace(newContents, "<KUBEFIRST_STATE_STORE_BUCKET>", tokens.KubefirstStateStoreBucket, -1)
			newContents = strings.Replace(newContents, "<KUBEFIRST_TEAM>", tokens.KubefirstTeam, -1)
			newContents = strings.Replace(newContents, "<KUBEFIRST_VERSION>", configs.K1Version, -1)

			newContents = strings.Replace(newContents, "<ARGOCD_INGRESS_URL>", tokens.ArgoCDIngressURL, -1)
			newContents = strings.Replace(newContents, "<ARGOCD_INGRESS_NO_HTTP_URL>", tokens.ArgoCDIngressNoHTTPSURL, -1)
			newContents = strings.Replace(newContents, "<ARGO_WORKFLOWS_INGRESS_URL>", tokens.ArgoWorkflowsIngressURL, -1)
			newContents = strings.Replace(newContents, "<ARGO_WORKFLOWS_INGRESS_NO_HTTPS_URL>", tokens.ArgoWorkflowsIngressNoHTTPSURL, -1)
			newContents = strings.Replace(newContents, "<ATLANTIS_INGRESS_URL>", tokens.AtlantisIngressURL, -1)
			newContents = strings.Replace(newContents, "<ATLANTIS_INGRESS_NO_HTTPS_URL>", tokens.AtlantisIngressNoHTTPSURL, -1)
			newContents = strings.Replace(newContents, "<CHARTMUSEUM_INGRESS_URL>", tokens.ChartMuseumIngressURL, -1)
			newContents = strings.Replace(newContents, "<VAULT_INGRESS_URL>", tokens.VaultIngressURL, -1)
			newContents = strings.Replace(newContents, "<VAULT_INGRESS_NO_HTTPS_URL>", tokens.VaultIngressNoHTTPSURL, -1)
			newContents = strings.Replace(newContents, "<VOUCH_INGRESS_URL>", tokens.VouchIngressURL, -1)

			newContents = strings.Replace(newContents, "<GIT_DESCRIPTION>", tokens.GitDescription, -1)
			newContents = strings.Replace(newContents, "<GIT_NAMESPACE>", tokens.GitNamespace, -1)
			newContents = strings.Replace(newContents, "<GIT_PROVIDER>", tokens.GitProvider, -1)
			newContents = strings.Replace(newContents, "<GIT_RUNNER>", tokens.GitRunner, -1)
			newContents = strings.Replace(newContents, "<GIT_RUNNER_DESCRIPTION>", tokens.GitRunnerDescription, -1)
			newContents = strings.Replace(newContents, "<GIT_RUNNER_NS>", tokens.GitRunnerNS, -1)
			newContents = strings.Replace(newContents, "<GIT_URL>", tokens.GitopsRepoGitURL, -1)

			newContents = strings.Replace(newContents, "<GITHUB_HOST>", tokens.GitHubHost, -1)
			newContents = strings.Replace(newContents, "<GITHUB_OWNER>", tokens.GitHubOwner, -1)
			newContents = strings.Replace(newContents, "<GITHUB_USER>", tokens.GitHubUser, -1)

			newContents = strings.Replace(newContents, "<GITOPS_REPO_ATLANTIS_WEBHOOK_URL>", tokens.GitOpsRepoAtlantisWebhookURL, -1)
			newContents = strings.Replace(newContents, "<GITOPS_REPO_GIT_URL>", tokens.GitOpsRepoGitURL, -1)
			newContents = strings.Replace(newContents, "<GITOPS_REPO_NO_HTTPS_URL>", tokens.GitOpsRepoNoHTTPSURL, -1)

			newContents = strings.Replace(newContents, "<METAPHOR_DEVELPOMENT_INGRESS_URL>", metaphorDevelopmentIngressURL, -1)
			newContents = strings.Replace(newContents, "<METAPHOR_DEVELOPMENT_INGRESS_NO_HTTPS_URL>", metaphorDevelopmentIngressNoHttpsURL, -1)
			newContents = strings.Replace(newContents, "<METAPHOR_PRODUCTION_INGRESS_URL>", metaphorProductionIngressURL, -1)
			newContents = strings.Replace(newContents, "<METAPHOR_PRODUCTION_INGRESS_NO_HTTPS_URL>", metaphorProductionIngressNoHttpsURL, -1)
			newContents = strings.Replace(newContents, "<METAPHOR_STAGING_INGRESS_URL>", metaphorStagingIngressURL, -1)
			newContents = strings.Replace(newContents, "<METAPHOR_STAGING_INGRESS_NO_HTTPS_URL>", metaphorStagingIngressNoHttpsURL, -1)

			err = ioutil.WriteFile(path, []byte(newContents), 0)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// detokenizeGithubMetaphor - Translate tokens by values on a given path
func DetokenizeGitMetaphor(path string, tokens *MetaphorTokenValues) error {

	err := filepath.Walk(path, detokenizeGitopsMetaphor(path, tokens))
	if err != nil {
		return err
	}

	return nil
}

func detokenizeGitopsMetaphor(metaphorDir string, tokens *MetaphorTokenValues) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !!fi.IsDir() {
			return nil
		}

		// var matched bool
		matched, err := filepath.Match("*", fi.Name())
		if matched {
			read, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			// todo reduce to terraform tokens by moving to helm chart?
			newContents := string(read)
			newContents = strings.Replace(newContents, "<METAPHOR_DEVELOPMENT_INGRESS_URL>", tokens.MetaphorDevelopmentIngressURL, -1)
			newContents = strings.Replace(newContents, "<METAPHOR_STAGING_INGRESS_URL>", tokens.MetaphorStagingIngressURL, -1)
			newContents = strings.Replace(newContents, "<METAPHOR_PRODUCTION_INGRESS_URL>", tokens.MetaphorProductionIngressURL, -1)
			newContents = strings.Replace(newContents, "<CONTAINER_REGISTRY_URL>", tokens.ContainerRegistryURL, -1) // todo need to fix metaphor repo names
			newContents = strings.Replace(newContents, "<DOMAIN_NAME>", tokens.DomainName, -1)
			newContents = strings.Replace(newContents, "<CLOUD_REGION>", tokens.CloudRegion, -1)
			newContents = strings.Replace(newContents, "<CLUSTER_NAME>", tokens.ClusterName, -1)

			err = ioutil.WriteFile(path, []byte(newContents), 0)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
