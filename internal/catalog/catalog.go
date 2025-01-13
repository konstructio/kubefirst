/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package catalog

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	git "github.com/google/go-github/v52/github"
	apiTypes "github.com/konstructio/kubefirst-api/pkg/types"
	"gopkg.in/yaml.v3"
)

const (
	KubefirstGitHubOrganization      = "kubefirst"
	KubefirstGitopsCatalogRepository = "gitops-catalog"
	basePath                         = "/"
)

type GitHubClient struct {
	Client *git.Client
}

// NewGitHub instantiates an unauthenticated GitHub client
func NewGitHub() *git.Client {
	return git.NewClient(nil)
}

func ReadActiveApplications() (*apiTypes.GitopsCatalogApps, error) {
	gh := GitHubClient{
		Client: NewGitHub(),
	}

	activeContent, err := gh.ReadGitopsCatalogRepoContents()
	if err != nil {
		return nil, fmt.Errorf("error retrieving gitops catalog repository content: %w", err)
	}

	index, err := gh.ReadGitopsCatalogIndex(activeContent)
	if err != nil {
		return nil, fmt.Errorf("error retrieving gitops catalog index content: %w", err)
	}

	var out apiTypes.GitopsCatalogApps

	err = yaml.Unmarshal(index, &out)
	if err != nil {
		return nil, fmt.Errorf("error retrieving gitops catalog applications: %w", err)
	}

	return &out, nil
}

func ValidateCatalogApps(catalogApps string) ([]apiTypes.GitopsCatalogApp, error) {
	if catalogApps == "" {
		// No catalog apps to install
		return nil, nil
	}

	apps, err := ReadActiveApplications()
	if err != nil {
		return nil, err
	}

	items := strings.Split(catalogApps, ",")
	gitopsCatalogapps := make([]apiTypes.GitopsCatalogApp, 0, len(items))
	for _, app := range items {
		found := false

		for _, catalogApp := range apps.Apps {
			if app == catalogApp.Name {
				found = true

				for pos, secret := range catalogApp.SecretKeys {
					secretValue := os.Getenv(secret.Env)
					if secretValue == "" {
						return nil, fmt.Errorf("your %q environment variable is not set for %q catalog application. Please set and try again", secret.Env, app)
					}

					secret.Value = secretValue
					catalogApp.SecretKeys[pos] = secret
				}

				for pos, config := range catalogApp.ConfigKeys {
					configValue := os.Getenv(config.Env)
					if configValue == "" {
						return nil, fmt.Errorf("your %q environment variable is not set for %q catalog application. Please set and try again", config.Env, app)
					}

					config.Value = configValue
					catalogApp.ConfigKeys[pos] = config
				}

				gitopsCatalogapps = append(gitopsCatalogapps, catalogApp)

				break
			}
		}

		if !found {
			return nil, fmt.Errorf("catalog app is not supported: %q", app)
		}
	}

	return gitopsCatalogapps, nil
}

func (gh *GitHubClient) ReadGitopsCatalogRepoContents() ([]*git.RepositoryContent, error) {
	_, directoryContent, _, err := gh.Client.Repositories.GetContents(
		context.Background(),
		KubefirstGitHubOrganization,
		KubefirstGitopsCatalogRepository,
		basePath,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error retrieving gitops catalog repository contents: %w", err)
	}

	return directoryContent, nil
}

// ReadGitopsCatalogIndex reads the gitops catalog repository index
func (gh *GitHubClient) ReadGitopsCatalogIndex(contents []*git.RepositoryContent) ([]byte, error) {
	for _, content := range contents {
		if *content.Type == "file" && *content.Name == "index.yaml" {
			b, err := gh.readFileContents(content)
			if err != nil {
				return nil, fmt.Errorf("error reading index.yaml file: %w", err)
			}
			return b, nil
		}
	}

	return nil, fmt.Errorf("index.yaml not found in gitops catalog repository")
}

// readFileContents parses the contents of a file in a GitHub repository
func (gh *GitHubClient) readFileContents(content *git.RepositoryContent) ([]byte, error) {
	rc, _, err := gh.Client.Repositories.DownloadContents(
		context.Background(),
		KubefirstGitHubOrganization,
		KubefirstGitopsCatalogRepository,
		*content.Path,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error downloading contents of %q: %w", *content.Path, err)
	}
	defer rc.Close()

	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("error reading contents of %q: %w", *content.Path, err)
	}

	return b, nil
}
