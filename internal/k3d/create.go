/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/pkg"
)

const (
	// https://hub.docker.com/r/rancher/k3s/tags?page=1&name=v1.23
	k3dImageTag string = "v1.26.3-k3s1"
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
		"--image", fmt.Sprintf("rancher/k3s:%s", k3dImageTag),
		"--agents", "3",
		"--agents-memory", "1024m",
		"--registry-create", "k3d-"+clusterName+"-registry",
		"--k3s-arg", `--kubelet-arg=eviction-hard=imagefs.available<1%,nodefs.available<1%@agent:*`,
		"--k3s-arg", `--kubelet-arg=eviction-minimum-reclaim=imagefs.available=1%,nodefs.available=1%@agent:*`,
		"--volume", volumeDir+":/var/lib/rancher/k3s/storage@all",
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
		return fmt.Errorf("error updating config")
	}

	return nil
}

// should tokens be a *GitopsTokenValues? does it matter
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
	gitopsTokens *GitopsTokenValues,
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
	detokenizeGitGitops(gitopsDir, gitopsTokens)
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
	detokenizeGitMetaphor(metaphorDir, metaphorTokens)
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
	// commit after metaphor content has been removed from gitops
	err = gitClient.Commit(gitopsRepo, "committing initial detokenized gitops-template repo content")
	if err != nil {
		return err
	}

	return nil
}

func PostRunPrepareGitopsRepository(clusterName string,
	//destinationGitopsRepoGitURL string,
	gitopsDir string,
	//gitopsRepo *git.Repository,
	tokens *GitopsTokenValues,
) error {

	err := postRunDetokenizeGitGitops(gitopsDir, tokens)
	if err != nil {
		return err
	}
	return nil
}
