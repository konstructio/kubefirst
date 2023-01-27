package pkg

import (
	"fmt"
	"os"
	"strings"

	cp "github.com/otiai10/copy"
	"github.com/rs/zerolog/log"
)

func AdjustGitopsTemplateContent(cloudProvider, clusterName, clusterType, gitProvider, k1DirPath, gitopsRepoPath string) error {

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

	//* copy civo-github/* $HOME/.k1/gitops/
	driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoPath, cloudProvider, gitProvider)
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
	err := cp.Copy(driverContent, fmt.Sprintf("%s", gitopsRepoPath), opt)
	if err != nil {
		log.Info().Msgf("Error populating gitops repository with %s setup: %s", fmt.Sprintf("%s/%s-%s/%s-cluster-template", gitopsRepoPath, cloudProvider, gitProvider, clusterType), err)
		return err
	}

	//* copy civo-github/mgmt-cluster-template/* registry/kubefirst
	clusterContent := fmt.Sprintf("%s/%s-cluster-template", gitopsRepoPath, clusterType)
	err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoPath, clusterName), opt)
	if err != nil {
		log.Info().Msgf("Error populating gitops repository with %s setup: %s", fmt.Sprintf("%s/%s-%s/%s-cluster-template", gitopsRepoPath, cloudProvider, gitProvider, clusterType), err)
		return err
	}

	ciFolderContent := fmt.Sprintf("%s/argo-workflows", gitopsRepoPath)
	err = cp.Copy(ciFolderContent, fmt.Sprintf("%s/argo-workflows", k1DirPath), opt)
	if err != nil {
		log.Info().Msgf("Error populating gitops repository with %s setup: %s", fmt.Sprintf("%s/%s-%s/%s-cluster-template", gitopsRepoPath, cloudProvider, gitProvider, clusterType), err)
		return err
	}

	os.RemoveAll(driverContent)
	os.RemoveAll(clusterContent)
	os.RemoveAll(ciFolderContent)
	return nil
}
