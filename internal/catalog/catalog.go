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

	"github.com/rs/zerolog/log"
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

func ReadActiveApplications(ctx context.Context) (apiTypes.GitopsCatalogApps, error) {
	gh := GitHubClient{
		Client: NewGitHub(),
	}

	activeContent, err := gh.ReadGitopsCatalogRepoContents(ctx)
	if err != nil {
		return apiTypes.GitopsCatalogApps{}, fmt.Errorf("error retrieving gitops catalog repository content: %w", err)
	}

	index, err := gh.ReadGitopsCatalogIndex(ctx, activeContent)
	if err != nil {
		return apiTypes.GitopsCatalogApps{}, fmt.Errorf("error retrieving gitops catalog index content: %w", err)
	}

	var out apiTypes.GitopsCatalogApps

	err = yaml.Unmarshal(index, &out)
	if err != nil {
		return apiTypes.GitopsCatalogApps{}, fmt.Errorf("error retrieving gitops catalog applications: %w", err)
	}

	return out, nil
}

func ValidateCatalogApps(ctx context.Context, catalogApps string) (bool, []apiTypes.GitopsCatalogApp, error) {
	items := strings.Split(catalogApps, ",")

	gitopsCatalogapps := []apiTypes.GitopsCatalogApp{}
	if catalogApps == "" {
		return true, gitopsCatalogapps, nil
	}

	apps, err := ReadActiveApplications(ctx)
	if err != nil {
		log.Error().Msgf("error getting gitops catalog applications: %s", err)
		return false, gitopsCatalogapps, err
	}

	for _, app := range items {
		found := false
		for _, catalogApp := range apps.Apps {
			if app == catalogApp.Name {
				found = true

				if catalogApp.SecretKeys != nil {
					for _, secret := range catalogApp.SecretKeys {
						secretValue := os.Getenv(secret.Env)

						if secretValue == "" {
							return false, gitopsCatalogapps, fmt.Errorf("your %q environment variable is not set for %q catalog application. Please set and try again", secret.Env, app)
						}

						secret.Value = secretValue
					}
				}

				if catalogApp.ConfigKeys != nil {
					for _, config := range catalogApp.ConfigKeys {
						configValue := os.Getenv(config.Env)
						if configValue == "" {
							return false, gitopsCatalogapps, fmt.Errorf("your %q environment variable is not set for %q catalog application. Please set and try again", config.Env, app)
						}
						config.Value = configValue
					}
				}

				gitopsCatalogapps = append(gitopsCatalogapps, catalogApp)

				break
			}
		}
		if !found {
			return false, gitopsCatalogapps, fmt.Errorf("catalog app is not supported: %q", app)
		}
	}

	return true, gitopsCatalogapps, nil
}

func (gh *GitHubClient) ReadGitopsCatalogRepoContents(ctx context.Context) ([]*git.RepositoryContent, error) {
	_, directoryContent, _, err := gh.Client.Repositories.GetContents(
		ctx,
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
func (gh *GitHubClient) ReadGitopsCatalogIndex(ctx context.Context, contents []*git.RepositoryContent) ([]byte, error) {
	for _, content := range contents {
		if *content.Type == "file" && *content.Name == "index.yaml" {
			b, err := gh.readFileContents(ctx, content)
			if err != nil {
				return nil, fmt.Errorf("error reading index.yaml file: %w", err)
			}
			return b, nil
		}
	}

	return nil, fmt.Errorf("index.yaml not found in gitops catalog repository")
}

// readFileContents parses the contents of a file in a GitHub repository
func (gh *GitHubClient) readFileContents(ctx context.Context, content *git.RepositoryContent) ([]byte, error) {
	rc, _, err := gh.Client.Repositories.DownloadContents(
		ctx,
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
