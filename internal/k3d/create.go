package k3d

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/pkg"
)

// ClusterCreate create an k3d cluster
func ClusterCreate(clusterName string, k1Dir string, k3dClient string, kubeconfig string) error {
	log.Info().Msg("creating K3d cluster...")

	volumeDir := fmt.Sprintf("%s/minio-storage", k1Dir)
	if _, err := os.Stat(volumeDir); os.IsNotExist(err) {
		err := os.MkdirAll(volumeDir, os.ModePerm)
		if err != nil {
			log.Info().Msgf("%s directory already exists, continuing", volumeDir)
		}
	}
	_, _, err := pkg.ExecShellReturnStrings(k3dClient, "cluster", "create",
		clusterName,
		"--agents", "3",
		"--agents-memory", "1024m",
		"--registry-create", "k3d-"+clusterName+"-registry:63630",
		"--k3s-arg", `--kubelet-arg=eviction-hard=imagefs.available<1%,nodefs.available<1%@agent:*`,
		"--k3s-arg", `--kubelet-arg=eviction-minimum-reclaim=imagefs.available=1%,nodefs.available=1%@agent:*`,
		"--port", "80:80@loadbalancer",
		"--volume", volumeDir+":/tmp/minio-storage",
		"--port", "443:443@loadbalancer")

	if err != nil {
		log.Info().Msg("error creating k3d cluster")
		return err
	}

	time.Sleep(20 * time.Second)

	kConfigString, _, err := pkg.ExecShellReturnStrings(k3dClient, "kubeconfig", "get", clusterName)
	if err != nil {
		return err
	}

	err = os.WriteFile(kubeconfig, []byte(kConfigString), 0644)
	if err != nil {
		log.Error().Err(err).Msg("error updating config")
		return errors.New("error updating config")
	}

	return nil
}

func PrepareGitopsRepository(clusterName string,
	clusterType string,
	destinationGitopsRepoGitURL string,
	gitopsDir string,
	gitopsTemplateBranch string,
	gitopsTemplateURL string,
	k1Dir string,
	tokens *GitopsTokenValues,
) error {

	gitopsRepo, err := gitClient.CloneRefSetMain(gitopsTemplateBranch, gitopsDir, gitopsTemplateURL)
	if err != nil {
		log.Info().Msgf("error opening repo at: %s", gitopsDir)
	}
	log.Info().Msg("gitops repository clone complete")

	err = k3dGithubAdjustGitopsTemplateContent(CloudProvider, clusterName, clusterType, GitProvider, k1Dir, gitopsDir)
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

func PrepareMetaphorRepository(
	destinationMetaphorRepoGitURL string,
	k1Dir string,
	metaphorDir string,
	metaphorTemplateBranch string,
	metaphorTemplateURL string,
	tokens *MetaphorTokenValues,
) error {

	log.Info().Msg("generating your new metaphor-frontend repository")
	metaphorRepo, err := gitClient.CloneRefSetMain(metaphorTemplateBranch, metaphorDir, metaphorTemplateURL)
	if err != nil {
		log.Info().Msgf("error opening repo at: %s", metaphorDir)
	}

	log.Info().Msg("metaphor repository clone complete")

	err = k3dGithubAdjustMetaphorTemplateContent(GitProvider, k1Dir, metaphorDir)
	if err != nil {
		return err
	}

	detokenizeGithubMetaphor(metaphorDir, tokens)

	err = gitClient.AddRemote(destinationMetaphorRepoGitURL, GitProvider, metaphorRepo)
	if err != nil {
		return err
	}

	err = gitClient.Commit(metaphorRepo, "committing detokenized metaphor-frontend-template repo content")
	if err != nil {
		return err
	}

	if err != nil {
		log.Panic().Msgf("error pushing detokenized gitops repository to remote %s", destinationMetaphorRepoGitURL)
	}

	return nil
}
