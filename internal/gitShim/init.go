/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gitShim

import (
	"fmt"

	"github.com/kubefirst/runtime/pkg/github"
	"github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/rs/zerolog/log"
)

type GitInitParameters struct {
	GitProvider  string
	GitToken     string
	GitOwner     string
	Repositories []string
	Teams        []string
}

// InitializeGitProvider
func InitializeGitProvider(p *GitInitParameters) error {
	switch p.GitProvider {
	case "github":
		githubSession := github.New(p.GitToken)
		newRepositoryExists := false
		// todo hoist to globals
		errorMsg := "the following repositories must be removed before continuing with your kubefirst installation.\n\t"

		for _, repositoryName := range p.Repositories {
			responseStatusCode := githubSession.CheckRepoExists(p.GitOwner, repositoryName)

			// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#get-a-repository
			repositoryExistsStatusCode := 200
			repositoryDoesNotExistStatusCode := 404

			if responseStatusCode == repositoryExistsStatusCode {
				log.Info().Msgf("repository https://github.com/%s/%s exists", p.GitOwner, repositoryName)
				errorMsg = errorMsg + fmt.Sprintf("https://github.com/%s/%s\n\t", p.GitOwner, repositoryName)
				newRepositoryExists = true
			} else if responseStatusCode == repositoryDoesNotExistStatusCode {
				log.Info().Msgf("repository https://github.com/%s/%s does not exist, continuing", p.GitOwner, repositoryName)
			}
		}
		if newRepositoryExists {
			return fmt.Errorf(errorMsg)
		}

		newTeamExists := false
		errorMsg = "the following teams must be removed before continuing with your kubefirst installation.\n\t"

		for _, teamName := range p.Teams {
			responseStatusCode := githubSession.CheckTeamExists(p.GitOwner, teamName)

			// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#get-a-team-by-name
			teamExistsStatusCode := 200
			teamDoesNotExistStatusCode := 404

			if responseStatusCode == teamExistsStatusCode {
				log.Info().Msgf("team https://github.com/%s/%s exists", p.GitOwner, teamName)
				errorMsg = errorMsg + fmt.Sprintf("https://github.com/orgs/%s/teams/%s\n\t", p.GitOwner, teamName)
				newTeamExists = true
			} else if responseStatusCode == teamDoesNotExistStatusCode {
				log.Info().Msgf("https://github.com/orgs/%s/teams/%s does not exist, continuing", p.GitOwner, teamName)
			}
		}
		if newTeamExists {
			return fmt.Errorf(errorMsg)
		}
	case "gitlab":
		gitlabClient, err := gitlab.NewGitLabClient(p.GitToken, p.GitOwner)
		if err != nil {
			return err
		}

		// Check for existing base projects
		projects, err := gitlabClient.GetProjects()
		if err != nil {
			log.Fatal().Msgf("couldn't get gitlab projects: %s", err)
		}
		for _, repositoryName := range p.Repositories {
			for _, project := range projects {
				if project.Name == repositoryName {
					return fmt.Errorf("project %s already exists and will need to be deleted before continuing", repositoryName)
				}
			}
		}

		// Check for existing base projects
		// Save for detokenize
		subgroups, err := gitlabClient.GetSubGroups()
		if err != nil {
			log.Fatal().Msgf("couldn't get gitlab subgroups for group %s: %s", p.GitOwner, err)
		}
		for _, teamName := range p.Repositories {
			for _, sg := range subgroups {
				if sg.Name == teamName {
					return fmt.Errorf("subgroup %s already exists and will need to be deleted before continuing", teamName)
				}
			}
		}
	}

	return nil
}
