/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gitShim

import (
	"fmt"
	"net/http"
	"os"

	"github.com/kubefirst/kubefirst-api/pkg/handlers"
	"github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/kubefirst/internal/progress"
	"github.com/kubefirst/runtime/pkg/github"
	"github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/services"
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
	progress.AddStep("Validate git environment")

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

	progress.CompleteStep("Validate git environment")

	return nil
}

func ValidateGitCredentials(gitProviderFlag string, githubOrgFlag string, gitlabGroupFlag string) (types.GitAuth, error) {
	progress.AddStep("Validate git credentials")
	gitAuth := types.GitAuth{}

	// Switch based on git provider, set params
	switch gitProviderFlag {
	case "github":
		if githubOrgFlag == "" {
			return gitAuth, fmt.Errorf("please provide a github organization using the --github-org flag")
		}
		if os.Getenv("GITHUB_TOKEN") == "" {
			return gitAuth, fmt.Errorf("your GITHUB_TOKEN is not set. Please set and try again")
		}

		gitAuth.Owner = githubOrgFlag
		gitAuth.Token = os.Getenv("GITHUB_TOKEN")

		// Verify token scopes
		err := github.VerifyTokenPermissions(gitAuth.Token)
		if err != nil {
			return gitAuth, err
		}

		// Handle authorization checks
		httpClient := http.DefaultClient
		gitHubService := services.NewGitHubService(httpClient)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)

		// get github data to set user based on the provided token
		log.Info().Msg("verifying github authentication")
		githubUser, err := gitHubHandler.GetGitHubUser(gitAuth.Token)
		if err != nil {
			return gitAuth, err
		}

		gitAuth.User = githubUser
		viper.Set("github.user", githubUser)
		err = viper.WriteConfig()
		if err != nil {
			return gitAuth, err
		}
		err = gitHubHandler.CheckGithubOrganizationPermissions(gitAuth.Token, githubOrgFlag, githubUser)
		if err != nil {
			return gitAuth, err
		}
		viper.Set("flags.github-owner", githubOrgFlag)
		viper.WriteConfig()
	case "gitlab":
		if gitlabGroupFlag == "" {
			return gitAuth, fmt.Errorf("please provide a gitlab group using the --gitlab-group flag")
		}
		if os.Getenv("GITLAB_TOKEN") == "" {
			return gitAuth, fmt.Errorf("your GITLAB_TOKEN is not set. please set and try again")
		}

		gitAuth.Token = os.Getenv("GITLAB_TOKEN")

		// Verify token scopes
		err := gitlab.VerifyTokenPermissions(gitAuth.Token)
		if err != nil {
			return gitAuth, err
		}

		gitlabClient, err := gitlab.NewGitLabClient(gitAuth.Token, gitlabGroupFlag)
		if err != nil {
			return gitAuth, err
		}

		gitAuth.Owner = gitlabClient.ParentGroupPath
		cGitlabOwnerGroupID := gitlabClient.ParentGroupID
		log.Info().Msgf("set gitlab owner to %s", gitAuth.Owner)

		// Get authenticated user's name
		user, _, err := gitlabClient.Client.Users.CurrentUser()
		if err != nil {
			return gitAuth, fmt.Errorf("unable to get authenticated user info - please make sure GITLAB_TOKEN env var is set %s", err)
		}
		gitAuth.User = user.Username

		viper.Set("flags.gitlab-owner", gitlabGroupFlag)
		viper.Set("flags.gitlab-owner-group-id", cGitlabOwnerGroupID)
		viper.WriteConfig()
	default:
		log.Error().Msgf("invalid git provider option")
	}

	progress.CompleteStep("Validate git credentials")

	return gitAuth, nil
}
