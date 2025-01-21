/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gitShim //nolint:revive // allowed during refactoring

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/konstructio/kubefirst-api/pkg/github"
	"github.com/konstructio/kubefirst-api/pkg/gitlab"
	"github.com/konstructio/kubefirst-api/pkg/handlers"
	"github.com/konstructio/kubefirst-api/pkg/services"
	"github.com/konstructio/kubefirst-api/pkg/types"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
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
		errorMsg := "the following repositories must be removed before continuing with your Kubefirst installation.\n\t"

		for _, repositoryName := range p.Repositories {
			responseStatusCode := githubSession.CheckRepoExists(p.GitOwner, repositoryName)

			repositoryExistsStatusCode := 200
			repositoryDoesNotExistStatusCode := 404

			if responseStatusCode == repositoryExistsStatusCode {
				log.Info().Msgf("repository %q exists", fmt.Sprintf("https://github.com/%s/%s", p.GitOwner, repositoryName))
				errorMsg += fmt.Sprintf("https://github.com/%s/%s\n\t", p.GitOwner, repositoryName)
				newRepositoryExists = true
			} else if responseStatusCode == repositoryDoesNotExistStatusCode {
				log.Info().Msgf("repository %q does not exist, continuing", fmt.Sprintf("https://github.com/%s/%s", p.GitOwner, repositoryName))
			}
		}
		if newRepositoryExists {
			return errors.New(errorMsg)
		}

		newTeamExists := false
		errorMsg = "the following teams must be removed before continuing with your Kubefirst installation.\n\t"

		for _, teamName := range p.Teams {
			responseStatusCode := githubSession.CheckTeamExists(p.GitOwner, teamName)

			// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#get-a-team-by-name
			teamExistsStatusCode := 200
			teamDoesNotExistStatusCode := 404

			if responseStatusCode == teamExistsStatusCode {
				log.Info().Msgf("team %q exists", fmt.Sprintf("https://github.com/%s/%s", p.GitOwner, teamName))
				errorMsg += fmt.Sprintf("https://github.com/orgs/%s/teams/%s\n\t", p.GitOwner, teamName)
				newTeamExists = true
			} else if responseStatusCode == teamDoesNotExistStatusCode {
				log.Info().Msgf("team %q does not exist, continuing", fmt.Sprintf("https://github.com/orgs/%s/teams/%s", p.GitOwner, teamName))
			}
		}
		if newTeamExists {
			return errors.New(errorMsg)
		}
	case "gitlab":
		gitlabClient, err := gitlab.NewGitLabClient(p.GitToken, p.GitOwner)
		if err != nil {
			return fmt.Errorf("error creating GitLab client: %w", err)
		}

		projects, err := gitlabClient.GetProjects()
		if err != nil {
			return fmt.Errorf("couldn't get GitLab projects: %w", err)
		}
		for _, repositoryName := range p.Repositories {
			for _, project := range projects {
				if project.Name == repositoryName {
					return fmt.Errorf("project %q already exists and will need to be deleted before continuing", repositoryName)
				}
			}
		}

		subgroups, err := gitlabClient.GetSubGroups()
		if err != nil {
			return fmt.Errorf("couldn't get GitLab subgroups for group %q: %w", p.GitOwner, err)
		}
		for _, teamName := range p.Teams {
			for _, sg := range subgroups {
				if sg.Name == teamName {
					return fmt.Errorf("subgroup %q already exists and will need to be deleted before continuing", teamName)
				}
			}
		}
	}

	return nil
}

func ValidateGitCredentials(gitProviderFlag, githubOrgFlag, gitlabGroupFlag string) (types.GitAuth, error) {

	// TODO: Handle for non-bubbletea
	// progress.AddStep("Validate git credentials")

	gitAuth := types.GitAuth{}

	switch gitProviderFlag {
	case "github":
		if githubOrgFlag == "" {
			return gitAuth, fmt.Errorf("please provide a GitHub organization using the --github-org flag")
		}
		if os.Getenv("GITHUB_TOKEN") == "" {
			return gitAuth, fmt.Errorf("your GITHUB_TOKEN is not set. Please set and try again")
		}

		gitAuth.Owner = githubOrgFlag
		gitAuth.Token = os.Getenv("GITHUB_TOKEN")

		err := github.VerifyTokenPermissions(gitAuth.Token)
		if err != nil {
			return gitAuth, fmt.Errorf("error verifying GitHub token permissions: %w", err)
		}

		httpClient := http.DefaultClient
		gitHubService := services.NewGitHubService(httpClient)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)

		log.Info().Msg("verifying GitHub authentication")
		githubUser, err := gitHubHandler.GetGitHubUser(gitAuth.Token)
		if err != nil {
			return gitAuth, fmt.Errorf("error getting GitHub user: %w", err)
		}

		gitAuth.User = githubUser
		viper.Set("github.user", githubUser)
		err = viper.WriteConfig()
		if err != nil {
			return gitAuth, fmt.Errorf("error writing GitHub config: %w", err)
		}
		err = gitHubHandler.CheckGithubOrganizationPermissions(gitAuth.Token, githubOrgFlag, githubUser)
		if err != nil {
			return gitAuth, fmt.Errorf("error checking GitHub organization permissions: %w", err)
		}
		viper.Set("flags.github-owner", githubOrgFlag)
		viper.WriteConfig()
	case "gitlab":
		if gitlabGroupFlag == "" {
			return gitAuth, fmt.Errorf("please provide a GitLab group using the --gitlab-group flag")
		}
		if os.Getenv("GITLAB_TOKEN") == "" {
			return gitAuth, fmt.Errorf("your GITLAB_TOKEN is not set. Please set and try again")
		}

		gitAuth.Token = os.Getenv("GITLAB_TOKEN")

		err := gitlab.VerifyTokenPermissions(gitAuth.Token)
		if err != nil {
			return gitAuth, fmt.Errorf("error verifying GitLab token permissions: %w", err)
		}

		gitlabClient, err := gitlab.NewGitLabClient(gitAuth.Token, gitlabGroupFlag)
		if err != nil {
			return gitAuth, fmt.Errorf("error creating GitLab client: %w", err)
		}

		gitAuth.Owner = gitlabClient.ParentGroupPath
		cGitlabOwnerGroupID := gitlabClient.ParentGroupID
		log.Info().Msgf("set GitLab owner to %q", gitAuth.Owner)

		user, _, err := gitlabClient.Client.Users.CurrentUser()
		if err != nil {
			return gitAuth, fmt.Errorf("unable to get authenticated user info - please make sure GITLAB_TOKEN env var is set: %w", err)
		}
		gitAuth.User = user.Username

		viper.Set("flags.gitlab-owner", gitlabGroupFlag)
		viper.Set("flags.gitlab-owner-group-id", cGitlabOwnerGroupID)
		viper.WriteConfig()
	default:
		log.Printf("invalid git provider option: %q", gitProviderFlag)
		return gitAuth, fmt.Errorf("invalid git provider: %q", gitProviderFlag)
	}

	// TODO: Handle for non-bubbletea
	// progress.CompleteStep("Validate git credentials")

	return gitAuth, nil
}
