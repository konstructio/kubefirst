package civo

import (
	"fmt"
	"os"
	"strings"

	cp "github.com/otiai10/copy"

	"github.com/rs/zerolog/log"
)

func CivoAdjustGitopsTemplateContent(cloudProvider, clusterName, clusterType, gitProvider, k1Dir, gitopsRepoDir string) error {

	// remove the unstructured driver content
	os.RemoveAll(gitopsRepoDir + "/components")
	os.RemoveAll(gitopsRepoDir + "/localhost")
	os.RemoveAll(gitopsRepoDir + "/registry")
	os.RemoveAll(gitopsRepoDir + "/validation")
	os.RemoveAll(gitopsRepoDir + "/terraform")
	os.RemoveAll(gitopsRepoDir + "/.gitignore")
	os.RemoveAll(gitopsRepoDir + "/LICENSE")
	os.RemoveAll(gitopsRepoDir + "/README.md")
	os.RemoveAll(gitopsRepoDir + "/atlantis.yaml")
	os.RemoveAll(gitopsRepoDir + "/logo.png")

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

	//* copy civo-(gitprovider)/* $HOME/.k1/gitops/
	driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, cloudProvider, gitProvider)
	err := cp.Copy(driverContent, gitopsRepoDir, opt)
	if err != nil {
		log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
		return err
	}

	//* copy $HOME/.k1/gitops/${clusterType}-cluster-template/* $HOME/.k1/gitops/registry/${clusterName}
	clusterContent := fmt.Sprintf("%s/%s-cluster-template", gitopsRepoDir, clusterType)
	err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoDir, clusterName), opt)
	if err != nil {
		log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
		return err
	}

	//* copy gitops/argo-workflows/* $HOME/.k1/argo-workflows
	ciFolderContent := fmt.Sprintf("%s/argo-workflows", gitopsRepoDir)
	err = cp.Copy(ciFolderContent, fmt.Sprintf("%s/argo-workflows", k1Dir), opt)
	if err != nil {
		log.Info().Msgf("Error populating gitops repository with %s setup: %s", fmt.Sprintf("%s/%s-%s/%s-cluster-template", gitopsRepoDir, cloudProvider, gitProvider, clusterType), err)
		return err
	}

	os.RemoveAll(driverContent)
	os.RemoveAll(clusterContent)
	os.RemoveAll(fmt.Sprintf("%s/workload-cluster-template", gitopsRepoDir)) // todo need to figure out a strategy to include this
	os.RemoveAll(ciFolderContent)
	return nil
}

func CivoAdjustMetaphorTemplateContent(gitProvider, k1Dir, metaphorRepoPath string) error {

	log.Info().Msg("removing old metaphor ci content")
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

	switch gitProvider {
	case "github":
		//* copy $HOME/.k1/argo-workflows/.github/* $HOME/.k1/metaphor/.github
		githubActionsFolderContent := fmt.Sprintf("%s/argo-workflows/.github", k1Dir)
		log.Info().Msgf("copying ci content: %s", githubActionsFolderContent)
		err := cp.Copy(githubActionsFolderContent, fmt.Sprintf("%s/.github", metaphorRepoPath), opt)
		if err != nil {
			log.Info().Msgf("error populating metaphor repository with %s: %s", githubActionsFolderContent, err)
			return err
		}
	case "gitlab":
		//* copy $HOME/.k1/argo-workflows/.gitlab-ci.yml/* $HOME/.k1/metaphor/.github
		gitlabCIContent := fmt.Sprintf("%s/argo-workflows/.gitlab-ci.yml", k1Dir)
		log.Info().Msgf("copying ci content: %s", gitlabCIContent)
		err := cp.Copy(gitlabCIContent, fmt.Sprintf("%s/.gitlab-ci.yml", metaphorRepoPath), opt)
		if err != nil {
			log.Info().Msgf("error populating metaphor repository with %s: %s", gitlabCIContent, err)
			return err
		}
	}

	//* copy $HOME/.k1/argo-workflows/.argo/* $HOME/.k1/metaphor/.argo
	argoWorkflowsFolderContent := fmt.Sprintf("%s/argo-workflows/.argo", k1Dir)
	err := cp.Copy(argoWorkflowsFolderContent, fmt.Sprintf("%s/.argo", metaphorRepoPath), opt)
	if err != nil {
		log.Info().Msgf("error populating metaphor repository with %s: %s", argoWorkflowsFolderContent, err)
		return err
	}

	return nil
}
