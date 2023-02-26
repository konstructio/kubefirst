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
	detokenizeDirectoryRecursively(gitopsDir+"/registry", tokens)
	if err != nil {
		return err
	}
	detokenizeDirectoryRecursively(gitopsDir+"/terraform", tokens)
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
