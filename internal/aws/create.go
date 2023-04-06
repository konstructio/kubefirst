/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"github.com/go-git/go-git/v5"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/rs/zerolog/log"
)

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
) error {

	//* clone the gitops-template repo
	gitopsRepo, err := gitClient.CloneRefSetMain(gitopsTemplateBranch, gitopsDir, gitopsTemplateURL)
	if err != nil {
		log.Info().Msgf("error opening repo at: %s", gitopsDir)
	}
	log.Info().Msg("gitops repository clone complete")

	//* adjust the content for the gitops repo
	err = AdjustGitopsRepo(CloudProvider, clusterName, clusterType, gitopsDir, gitProvider, k1Dir)
	if err != nil {
		return err
	}

	//* detokenize the gitops repo
	DetokenizeGitGitops(gitopsDir, gitopsTokens)
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

	metaphorRepo, _ := git.PlainOpen(metaphorDir)
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

	//* commit initial gitops-template content
	// need to wait for the metaphor content to be removed
	err = gitClient.Commit(gitopsRepo, "committing initial detokenized gitops-template repo content")
	if err != nil {
		return err
	}

	return nil
}
