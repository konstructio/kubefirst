/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package digitalocean

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/pkg"

	cp "github.com/otiai10/copy"
	"github.com/rs/zerolog/log"
)

func AdjustGitopsRepo(cloudProvider, clusterName, clusterType, gitopsRepoDir, gitProvider, k1Dir string, apexContentExists bool) error {

	//* clean up all other platforms
	for _, platform := range pkg.SupportedPlatforms {
		if platform != fmt.Sprintf("%s-%s", CloudProvider, gitProvider) {
			os.RemoveAll(gitopsRepoDir + "/" + platform)
		}
	}

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

	//* copy $cloudProvider-$gitProvider/* $HOME/.k1/gitops/
	driverContent := fmt.Sprintf("%s/%s-%s/", gitopsRepoDir, CloudProvider, gitProvider)
	err := cp.Copy(driverContent, gitopsRepoDir, opt)
	if err != nil {
		log.Info().Msgf("Error populating gitops repository with driver content: %s. error: %s", fmt.Sprintf("%s-%s", CloudProvider, gitProvider), err.Error())
		return err
	}
	os.RemoveAll(driverContent)

	//* copy $HOME/.k1/gitops/cluster-types/${clusterType}/* $HOME/.k1/gitops/registry/${clusterName}
	clusterContent := fmt.Sprintf("%s/cluster-types/%s", gitopsRepoDir, clusterType)

	// Remove apex content if apex content already exists
	if apexContentExists {
		log.Warn().Msgf("removing nginx-apex since apexContentExists was %v", apexContentExists)
		os.Remove(fmt.Sprintf("%s/nginx-apex.yaml", clusterContent))
		os.RemoveAll(fmt.Sprintf("%s/nginx-apex", clusterContent))
	} else {
		log.Warn().Msgf("will create nginx-apex since apexContentExists was %v", apexContentExists)
	}

	err = cp.Copy(clusterContent, fmt.Sprintf("%s/registry/%s", gitopsRepoDir, clusterName), opt)
	if err != nil {
		log.Info().Msgf("Error populating cluster content with %s. error: %s", clusterContent, err.Error())
		return err
	}
	os.RemoveAll(fmt.Sprintf("%s/cluster-types", gitopsRepoDir))
	os.RemoveAll(fmt.Sprintf("%s/services", gitopsRepoDir))

	return nil
}

func AdjustMetaphorRepo(destinationMetaphorRepoGitURL, gitopsRepoDir, gitProvider, k1Dir string) error {

	//* create ~/.k1/metaphor
	metaphorDir := fmt.Sprintf("%s/metaphor", k1Dir)
	os.Mkdir(metaphorDir, 0700)

	//* git init
	metaphorRepo, err := git.PlainInit(metaphorDir, false)
	if err != nil {
		return err
	}

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

	//* copy ci content
	switch gitProvider {
	case "github":
		//* copy $HOME/.k1/gitops/ci/.github/* $HOME/.k1/metaphor/.github
		githubActionsFolderContent := fmt.Sprintf("%s/gitops/ci/.github", k1Dir)
		log.Info().Msgf("copying github content: %s", githubActionsFolderContent)
		err := cp.Copy(githubActionsFolderContent, fmt.Sprintf("%s/.github", metaphorDir), opt)
		if err != nil {
			log.Info().Msgf("error populating metaphor repository with %s: %s", githubActionsFolderContent, err)
			return err
		}
	case "gitlab":
		//* copy $HOME/.k1/gitops/ci/.gitlab-ci.yml/* $HOME/.k1/metaphor/.github
		gitlabCIContent := fmt.Sprintf("%s/gitops/ci/.gitlab-ci.yml", k1Dir)
		log.Info().Msgf("copying gitlab content: %s", gitlabCIContent)
		err := cp.Copy(gitlabCIContent, fmt.Sprintf("%s/.gitlab-ci.yml", metaphorDir), opt)
		if err != nil {
			log.Info().Msgf("error populating metaphor repository with %s: %s", gitlabCIContent, err)
			return err
		}
	}

	//* metaphor app source
	metaphorContent := fmt.Sprintf("%s/metaphor", gitopsRepoDir)
	err = cp.Copy(metaphorContent, metaphorDir, opt)
	if err != nil {
		log.Info().Msgf("Error populating metaphor content with %s. error: %s", metaphorContent, err.Error())
		return err
	}

	//* copy $HOME/.k1/gitops/ci/.argo/* $HOME/.k1/metaphor/.argo
	argoWorkflowsFolderContent := fmt.Sprintf("%s/gitops/ci/.argo", k1Dir)
	log.Info().Msgf("copying argo workflows content: %s", argoWorkflowsFolderContent)
	err = cp.Copy(argoWorkflowsFolderContent, fmt.Sprintf("%s/.argo", metaphorDir), opt)
	if err != nil {
		log.Info().Msgf("error populating metaphor repository with %s: %s", argoWorkflowsFolderContent, err)
		return err
	}

	//* copy $HOME/.k1/gitops/metaphor/Dockerfile $HOME/.k1/metaphor/build/Dockerfile
	dockerfileContent := fmt.Sprintf("%s/Dockerfile", metaphorDir)
	os.Mkdir(metaphorDir+"/build", 0700)
	log.Info().Msgf("copying dockerfile content: %s", argoWorkflowsFolderContent)
	err = cp.Copy(dockerfileContent, fmt.Sprintf("%s/build/Dockerfile", metaphorDir), opt)
	if err != nil {
		log.Info().Msgf("error populating metaphor repository with %s: %s", argoWorkflowsFolderContent, err)
		return err
	}
	os.RemoveAll(fmt.Sprintf("%s/metaphor", gitopsRepoDir))

	//  add
	// commit
	err = gitClient.Commit(metaphorRepo, "committing initial detokenized metaphor repo content")
	if err != nil {
		return err
	}

	metaphorRepo, err = gitClient.SetRefToMainBranch(metaphorRepo)
	if err != nil {
		return err
	}

	// remove old git ref
	err = metaphorRepo.Storer.RemoveReference(plumbing.NewBranchReferenceName("master"))
	if err != nil {
		return fmt.Errorf("error removing previous git ref: %s", err)
	}
	// create remote
	_, err = metaphorRepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{destinationMetaphorRepoGitURL},
	})
	return nil
}

func PrepareGitRepositories(
	gitProvider string,
	clusterName string,
	clusterType string,
	destinationGitopsRepoGitURL string,
	gitopsDir string,
	gitopsTemplateBranch string,
	gitopsTemplateURL string,
	destinationMetaphorRepoGitURL string,
	k1Dir string,
	gitopsTokens *GitOpsDirectoryValues,
	metaphorDir string,
	metaphorTokens *MetaphorTokenValues,
	apexContentExists bool,
) error {

	//* clone the gitops-template repo
	gitopsRepo, err := gitClient.CloneRefSetMain(gitopsTemplateBranch, gitopsDir, gitopsTemplateURL)
	if err != nil {
		log.Info().Msgf("error opening repo at: %s", gitopsDir)
	}
	log.Info().Msg("gitops repository clone complete")

	//* adjust the content for the gitops repo
	err = AdjustGitopsRepo(CloudProvider, clusterName, clusterType, gitopsDir, gitProvider, k1Dir, apexContentExists)
	if err != nil {
		return err
	}

	//* detokenize the gitops repo
	DetokenizeGitGitops(gitopsDir, gitopsTokens)
	if err != nil {
		return err
	}

	//* commit initial gitops-template content
	err = gitClient.Commit(gitopsRepo, "committing initial detokenized gitops-template repo content")
	if err != nil {
		return err
	}

	//* add new remote
	err = gitClient.AddRemote(destinationGitopsRepoGitURL, gitProvider, gitopsRepo)
	if err != nil {
		return err
	}

	//! metaphor
	//* adjust the content for the gitops repo
	err = AdjustMetaphorRepo(destinationMetaphorRepoGitURL, gitopsDir, gitProvider, k1Dir)
	if err != nil {
		return err
	}

	//* detokenize the gitops repo
	DetokenizeGitMetaphor(metaphorDir, metaphorTokens)
	if err != nil {
		return err
	}

	metaphorRepo, err := git.PlainOpen(metaphorDir)
	//* commit initial gitops-template content
	err = gitClient.Commit(metaphorRepo, "committing initial detokenized metaphor repo content")
	if err != nil {
		return err
	}

	//* add new remote
	err = gitClient.AddRemote(destinationMetaphorRepoGitURL, gitProvider, metaphorRepo)
	if err != nil {
		return err
	}

	return nil
}
