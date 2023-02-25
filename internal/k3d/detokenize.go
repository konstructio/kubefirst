package k3d

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kubefirst/kubefirst/configs"
)

// detokenizeGitGitops - Translate tokens by values on a given path
func detokenizeGitGitops(path string, tokens *GitopsTokenValues) error {

	err := filepath.Walk(path, detokenizeGitops(path, tokens))
	if err != nil {
		return err
	}

	return nil
}

func detokenizeGitops(path string, tokens *GitopsTokenValues) filepath.WalkFunc {
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
			newContents = strings.Replace(newContents, "<ALERTS_EMAIL>", "your@email.com", -1) //
			newContents = strings.Replace(newContents, "<ARGO_CD_INGRESS_URL>", tokens.ArgocdIngressURL, -1)
			newContents = strings.Replace(newContents, "<ARGO_WORKFLOWS_INGRESS_URL>", tokens.ArgoWorkflowsIngressURL, -1)
			newContents = strings.Replace(newContents, "<ATLANTIS_ALLOW_LIST>", tokens.AtlantisAllowList, -1)
			newContents = strings.Replace(newContents, "<ATLANTIS_INGRESS_URL>", tokens.AtlantisIngressURL, -1)
			newContents = strings.Replace(newContents, "<CLUSTER_NAME>", tokens.ClusterName, -1)
			newContents = strings.Replace(newContents, "<DOMAIN_NAME>", DomainName, -1)
			newContents = strings.Replace(newContents, "<KUBEFIRST_VERSION>", configs.K1Version, -1)
			newContents = strings.Replace(newContents, "<METAPHOR_DEVELPOMENT_INGRESS_URL>", tokens.MetaphorDevelopmentIngressURL, -1)
			newContents = strings.Replace(newContents, "<METAPHOR_STAGING_INGRESS_URL>", tokens.MetaphorStagingIngressURL, -1)
			newContents = strings.Replace(newContents, "<METAPHOR_PRODUCTION_INGRESS_URL>", tokens.MetaphorProductionIngressURL, -1)
			newContents = strings.Replace(newContents, "<GITHUB_HOST>", tokens.GithubHost, -1)
			newContents = strings.Replace(newContents, "<GITHUB_OWNER>", tokens.GithubOwner, -1)
			newContents = strings.Replace(newContents, "<GITHUB_USER>", tokens.GithubUser, -1)
			newContents = strings.Replace(newContents, "<GITOPS_REPO_GIT_URL>", tokens.GitopsRepoGitURL, -1)
			newContents = strings.Replace(newContents, "<GITLAB_GROUP_NAME>", tokens.GitlabOwner, -1)
			newContents = strings.Replace(newContents, "<NGROK_HOST>", tokens.NgrokHost, -1)
			newContents = strings.Replace(newContents, "<VAULT_INGRESS_URL>", tokens.VaultIngressURL, -1)

			err = ioutil.WriteFile(path, []byte(newContents), 0)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// postRunDetokenizeGitGitops - Translate tokens by values on a given path
func postRunDetokenizeGitGitops(path string, tokens *GitopsTokenValues) error {

	err := filepath.Walk(path, postRunDetokenizeGitops(path, tokens))
	if err != nil {
		return err
	}

	return nil
}

func postRunDetokenizeGitops(path string, tokens *GitopsTokenValues) filepath.WalkFunc {
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

			//change Minio post cluster launch to cluster svc address
			newContents := string(read)
			newContents = strings.Replace(newContents, "http://minio.localdev.me", "http://minio.minio.svc.cluster.local:9000", -1)
			newContents = strings.Replace(newContents, "tfstate.tf", "terraform.tfstate", -1)

			err = ioutil.WriteFile(path, []byte(newContents), 0)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// detokenizeGitMetaphor - Translate tokens by values on a given path
func detokenizeGitMetaphor(path string, tokens *MetaphorTokenValues) error {

	err := filepath.Walk(path, detokenize(path, tokens))
	if err != nil {
		return err
	}

	return nil
}

func detokenize(metaphorDir string, tokens *MetaphorTokenValues) filepath.WalkFunc {
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
			newContents = strings.Replace(newContents, "<CONTAINER_REGISTRY>", tokens.ContainerRegistryURL, -1) // todo need to fix metaphor repo names
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
