package pkg

import (
	"fmt"
	"os"
	"strings"

	cp "github.com/otiai10/copy"
	"github.com/rs/zerolog/log"
)

func CivoGithubAdjustGitopsTemplateContent(cloudProvider, clusterName, clusterType, gitProvider, k1DirPath, gitopsRepoPath string) error {

	// remove the unstructured driver content
	os.RemoveAll(gitopsRepoPath + "/components")
	os.RemoveAll(gitopsRepoPath + "/localhost")
	os.RemoveAll(gitopsRepoPath + "/registry")
	os.RemoveAll(gitopsRepoPath + "/validation")
	os.RemoveAll(gitopsRepoPath + "/terraform")
	os.RemoveAll(gitopsRepoPath + "/.gitignore")
	os.RemoveAll(gitopsRepoPath + "/LICENSE")
	os.RemoveAll(gitopsRepoPath + "/README.md")
	os.RemoveAll(gitopsRepoPath + "/atlantis.yaml")
	os.RemoveAll(gitopsRepoPath + "/logo.png")

	//* copy options
	opt := cp.Options{
		Skip: func(src string) (bool, error) {
			if strings.HasSuffix(src, ".git") {
				return true, nil
			} else if strings.Index(src, "/.terraform") > 0 {
				return true, nil
			}
			//Add more stuff to be ignored here
			return false, nil

		},
	}

	//* copy civo-github/* $HOME/.k1/gitops/
	driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoPath, cloudProvider, gitProvider)
	err := cp.Copy(driverContent, gitopsRepoPath, opt)
	if err != nil {
		log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
		return err
	}

	//* copy gitops/$clusterType-cluster-template/* registry/$clusterName
	clusterContent := fmt.Sprintf("%s/%s-cluster-template", gitopsRepoPath, clusterType)
	err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoPath, clusterName), opt)
	if err != nil {
		log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
		return err
	}

	//* copy gitops/argo-workflows/* $HOME/.k1/argo-workflows
	ciFolderContent := fmt.Sprintf("%s/argo-workflows", gitopsRepoPath)
	err = cp.Copy(ciFolderContent, fmt.Sprintf("%s/argo-workflows", k1DirPath), opt)
	if err != nil {
		log.Info().Msgf("Error populating gitops repository with %s setup: %s", fmt.Sprintf("%s/%s-%s/%s-cluster-template", gitopsRepoPath, cloudProvider, gitProvider, clusterType), err)
		return err
	}

	//* rename file from `registry-mgmt.yaml` `registry-$clusterName.yaml`
	originalPath := fmt.Sprintf("%s/registry/%s/registry-mgmt.yaml", gitopsRepoPath, clusterName)
	newPath := fmt.Sprintf("%s/registry/%s/registry-%s.yaml", gitopsRepoPath, clusterName, clusterName)
	err = os.Rename(originalPath, newPath)
	if err != nil {
		log.Info().Msg(err.Error())
		return err
	}

	os.RemoveAll(driverContent)
	os.RemoveAll(clusterContent)
	os.RemoveAll(fmt.Sprintf("%s/workload-cluster-template", gitopsRepoPath)) // todo need to figure out a strategy to include this
	os.RemoveAll(ciFolderContent)
	return nil
}

func CivoGithubAdjustMetaphorTemplateContent(gitProvider, k1DirPath, metaphorRepoPath string) error {

	// remove the unstructured driver content
	os.RemoveAll(metaphorRepoPath + "/.argo")
	os.RemoveAll(metaphorRepoPath + "/.github")
	os.RemoveAll(metaphorRepoPath + "/.gitlab-ci.yml")

	//* copy options
	opt := cp.Options{
		Skip: func(src string) (bool, error) {
			if strings.HasSuffix(src, ".git") {
				return true, nil
			} else if strings.Index(src, "/.terraform") > 0 {
				return true, nil
			}
			//Add more stuff to be ignored here
			return false, nil

		},
	}

	//* copy $HOME/.k1/argo-workflows/.github/* $HOME/.k1/metaphor-frontend/.github
	githubActionsFolderContent := fmt.Sprintf("%s/argo-workflows/.github", k1DirPath)
	err := cp.Copy(githubActionsFolderContent, fmt.Sprintf("%s/.github", metaphorRepoPath), opt)
	if err != nil {
		log.Info().Msgf("Error metaphor repository with %s setup: %s", githubActionsFolderContent, err)
		return err
	}

	//* copy $HOME/.k1/argo-workflows/.argo/* $HOME/.k1/metaphor-frontend/.argo
	argoWorkflowsFolderContent := fmt.Sprintf("%s/argo-workflows/.argo", k1DirPath)
	err = cp.Copy(argoWorkflowsFolderContent, fmt.Sprintf("%s/.argo", metaphorRepoPath), opt)
	if err != nil {
		log.Info().Msgf("Error metaphor repository with %s setup: %s", argoWorkflowsFolderContent, err)
		return err
	}

	return nil
}
