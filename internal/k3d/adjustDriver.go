package k3d

import (
	"fmt"
	"os"
	"strings"

	cp "github.com/otiai10/copy"
	"github.com/rs/zerolog/log"
)

func k3dGithubAdjustGitopsTemplateContent(cloudProvider, clusterName, clusterType, gitProvider, k1Dir, gitopsRepoDir string) error {

	// remove the unstructured driver content
	os.RemoveAll(gitopsRepoDir + "/atlantis.yaml")
	os.RemoveAll(gitopsRepoDir + "/.gitignore")
	os.RemoveAll(gitopsRepoDir + "/components")
	os.RemoveAll(gitopsRepoDir + "/registry")
	os.RemoveAll(gitopsRepoDir + "/terraform")
	os.RemoveAll(gitopsRepoDir + "/validation")
	os.RemoveAll(gitopsRepoDir + "/LICENSE")
	os.RemoveAll(gitopsRepoDir + "/README.md")
	os.RemoveAll(gitopsRepoDir + "/logo.png")
	os.RemoveAll(gitopsRepoDir + "/civo-github")

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

	//* copy k3d-github/* $HOME/.k1/gitops/
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

func K3dGithubAdjustMetaphorTemplateContent(gitProvider, k1Dir, metaphorRepoPath string) error {

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
	githubActionsFolderContent := fmt.Sprintf("%s/argo-workflows/.github", k1Dir)
	err := cp.Copy(githubActionsFolderContent, fmt.Sprintf("%s/.github", metaphorRepoPath), opt)
	if err != nil {
		log.Info().Msgf("error populating metaphor repository with %s: %s", githubActionsFolderContent, err)
		return err
	}

	//* copy $HOME/.k1/argo-workflows/.argo/* $HOME/.k1/metaphor-frontend/.argo
	argoWorkflowsFolderContent := fmt.Sprintf("%s/argo-workflows/.argo", k1Dir)
	err = cp.Copy(argoWorkflowsFolderContent, fmt.Sprintf("%s/.argo", metaphorRepoPath), opt)
	if err != nil {
		log.Info().Msgf("error populating metaphor repository with %s: %s", argoWorkflowsFolderContent, err)
		return err
	}

	return nil
}
