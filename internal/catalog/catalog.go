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

	apiTypes "github.com/kubefirst/kubefirst-api/pkg/types"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
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

func ReadActiveApplications() (apiTypes.GitopsCatalogApps, error) {
	gh := GitHubClient{
		Client: NewGitHub(),
	}

	activeContent, err := gh.ReadGitopsCatalogRepoContents()
	if err != nil {
		return apiTypes.GitopsCatalogApps{}, fmt.Errorf("error retrieving gitops catalog repository content: %s", err)
	}

	index, err := gh.ReadGitopsCatalogIndex(activeContent)
	if err != nil {
		return apiTypes.GitopsCatalogApps{}, fmt.Errorf("error retrieving gitops catalog index content: %s", err)
	}

	var out apiTypes.GitopsCatalogApps

	err = yaml.Unmarshal(index, &out)
	if err != nil {
		return apiTypes.GitopsCatalogApps{}, fmt.Errorf("error retrieving gitops catalog applications: %s", err)
	}

	return out, nil
}

func ValidateCatalogApps(catalogApps string) (bool, []apiTypes.GitopsCatalogApp, error) {
	items := strings.Split(catalogApps, ",")

	gitopsCatalogapps := []apiTypes.GitopsCatalogApp{}
	if catalogApps == "" {
		return true, gitopsCatalogapps, nil
	}

	apps, err := ReadActiveApplications()
	if err != nil {
		log.Error().Msgf(fmt.Sprintf("Error getting gitops catalag applications: %s", err))
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
							return false, gitopsCatalogapps, fmt.Errorf("your %s environment variable is not set for %s catalog application. Please set and try again", secret.Env, app)
						}

						secret.Value = secretValue
					}
				}

				if catalogApp.ConfigKeys != nil {
					for _, config := range catalogApp.ConfigKeys {
						configValue := os.Getenv(config.Env)
						if configValue == "" {
							return false, gitopsCatalogapps, fmt.Errorf("your %s environment variable is not set for %s catalog application. Please set and try again", config.Env, app)
						}
						config.Value = configValue
					}
				}

				gitopsCatalogapps = append(gitopsCatalogapps, catalogApp)

				break
			}
		}
		if !found {
			return false, gitopsCatalogapps, fmt.Errorf(fmt.Sprintf("catalag app is not supported: %s", app))
		}
	}

	return true, gitopsCatalogapps, nil
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
		return nil, err
	}

	return directoryContent, nil
}

// ReadGitopsCatalogIndex reads the gitops catalog repository index
func (gh *GitHubClient) ReadGitopsCatalogIndex(contents []*git.RepositoryContent) ([]byte, error) {
	for _, content := range contents {
		switch *content.Type {
		case "file":
			switch *content.Name {
			case "index.yaml":
				b, err := gh.readFileContents(content)
				if err != nil {
					return b, err
				}
				return b, nil
			}
		}
	}

	return []byte{}, fmt.Errorf("index.yaml not found in gitops catalog repository")
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
		return []byte{}, err
	}
	defer rc.Close()

	b, err := io.ReadAll(rc)
	if err != nil {
		return []byte{}, err
	}

	return b, nil
}
