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

	//* commit initial gitops-template content
	// need to wait for the metaphor content to be removed
	err = gitClient.Commit(gitopsRepo, "committing initial detokenized gitops-template repo content")
	if err != nil {
		return err
	}

	return nil
}

func CheckForIamRoles(roles []string) error {
	awsClient := &Conf

	msg := "\n\nerror: the following role(s) exist and will create a collision\nwith the current terraform configuration.\n\n\t"

	// var rolesExist []string

	for _, roleName := range roles {

		role, err := awsClient.GetIamRole(roleName)
		if err != nil {
			return err
		}

		// rolesExist = append(rolesExist, *role.Role.RoleName)
		msg = msg + *role.Role.Arn + "\n\t"
	}
	msg = msg + "\nrun `kubefirst reset` with a unique `--cluster-name` to avoid collision or\n"
	msg = msg + "\n`kubefirst aws rm-roles --confirm`\nto remove them from AWS\n\n"

	return nil
}
