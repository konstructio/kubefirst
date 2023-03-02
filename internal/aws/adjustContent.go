package aws

import (
	"fmt"
	"os"
	"strings"

	cp "github.com/otiai10/copy"

	"github.com/rs/zerolog/log"
)

func adjustGitopsTemplateContent(cloudProvider, clusterName, clusterType, gitProvider, k1Dir, gitopsRepoDir string) error {

	var otherGitProvider string
	if gitProvider == "github" {
		otherGitProvider = "gitlab"
	} else if gitProvider == "gitlab" {
		otherGitProvider = "github"
	}

	// remove the other platforms driver content
	os.RemoveAll(gitopsRepoDir + fmt.Sprintf("/aws-%s", otherGitProvider))
	os.RemoveAll(gitopsRepoDir + "/civo-github")
	os.RemoveAll(gitopsRepoDir + "/civo-gitlab")
	os.RemoveAll(gitopsRepoDir + "/k3d-github")
	os.RemoveAll(gitopsRepoDir + "/k3d-gitlab")

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

	//* copy aws-github/* $HOME/.k1/gitops/
	driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, cloudProvider, gitProvider)
	err := cp.Copy(driverContent, gitopsRepoDir, opt)
	if err != nil {
		log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", cloudProvider, gitProvider), err.Error())
		return err
	}
	os.RemoveAll(driverContent)

	//* copy $HOME/.k1/gitops/.kubefirst/clusters/${clusterType}-template/* $HOME/.k1/gitops/registry/${clusterName}
	clusterContent := fmt.Sprintf("%s/.kubefirst/clusters/%s-template", gitopsRepoDir, clusterType)
	err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoDir, clusterName), opt)
	if err != nil {
		log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
		return err
	}

	return nil
}

// todo better name here
func adjustMetaphorTemplateContent(k1Dir, metaphorRepoPath string) error {

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

	//* copy $HOME/.k1/gitops/.kubefirst/ci/.github/* $HOME/.k1/metaphor-frontend/.github
	githubActionsFolderContent := fmt.Sprintf("%s/gitops/.kubefirst/ci/.github", k1Dir)
	log.Info().Msgf("copying ci content: %s", githubActionsFolderContent)
	err := cp.Copy(githubActionsFolderContent, fmt.Sprintf("%s/.github", metaphorRepoPath), opt)
	if err != nil {
		log.Info().Msgf("error populating metaphor repository with %s: %s", githubActionsFolderContent, err)
		return err
	}

	//* copy $HOME/.k1/gitops/.kubefirst/ci/.argo/* $HOME/.k1/metaphor-frontend/.argo
	argoWorkflowsFolderContent := fmt.Sprintf("%s/gitops/.kubefirst/ci/.argo", k1Dir)
	log.Info().Msgf("copying ci content: %s", argoWorkflowsFolderContent)
	err = cp.Copy(argoWorkflowsFolderContent, fmt.Sprintf("%s/.argo", metaphorRepoPath), opt)
	if err != nil {
		log.Info().Msgf("error populating metaphor repository with %s: %s", argoWorkflowsFolderContent, err)
		return err
	}

	return nil
}
