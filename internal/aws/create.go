package aws

import (
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/rs/zerolog/log"
)

func PrepareGitopsRepository(clusterName string,
	clusterType string,
	destinationGitopsRepoGitURL string,
	gitopsDir string,
	gitopsTemplateBranch string,
	gitopsTemplateURL string,
	k1Dir string,
	tokens *GitOpsDirectoryValues,
) error {

	gitopsRepo, err := gitClient.CloneRefSetMain(gitopsTemplateBranch, gitopsDir, gitopsTemplateURL)
	if err != nil {
		log.Info().Msgf("error opening repo at: %s", gitopsDir)
	}
	log.Info().Msg("gitops repository clone complete")

	err = adjustGitopsTemplateContent(CloudProvider, clusterName, clusterType, GitProvider, k1Dir, gitopsDir)
	if err != nil {
		return err
	}

	detokenizeGithubGitops(gitopsDir, tokens)
	if err != nil {
		return err
	}
	err = gitClient.AddRemote(destinationGitopsRepoGitURL, GitProvider, gitopsRepo)
	if err != nil {
		return err
	}

	err = gitClient.Commit(gitopsRepo, "committing initial detokenized gitops-template repo content")
	if err != nil {
		return err
	}
	return nil
}
