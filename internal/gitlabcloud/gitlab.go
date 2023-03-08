package gitlabcloud

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
)

func NewGitLabClient(token string) *gitlab.Client {
	git, err := gitlab.NewClient(token)
	if err != nil {
		fmt.Println(err)
	}

	return git
}

// AddSubGroupToGroup
func (gl *GitLabWrapper) AddSubGroupToGroup(subGroupID int, groupID int) error {
	group, resp, err := gl.Client.Groups.TransferSubGroup(subGroupID, &gitlab.TransferSubGroupOptions{
		GroupID: &groupID,
	})
	if err != nil {
		if resp.StatusCode == 400 {
			return errors.New("subgroup has already been added to group")
		}
		return err
	}
	log.Info().Msgf("subgroup %d added to group %s", subGroupID, group.Name)

	return nil
}

// CheckProjectExists
func (gl *GitLabWrapper) CheckProjectExists(projectName string) (bool, error) {
	allprojects, err := gl.GetProjects()
	if err != nil {
		return false, err
	}

	var exists bool = false
	for _, project := range allprojects {
		if project.Name == projectName {
			exists = true
		}
	}

	return exists, nil
}

// CreateSubGroup
func (gl *GitLabWrapper) CreateSubGroup(groupID int, groupName string) error {
	path := fmt.Sprintf("group-%s", groupName)
	group, _, err := gl.Client.Groups.CreateGroup(&gitlab.CreateGroupOptions{
		Name:     &groupName,
		Path:     &path,
		ParentID: &groupID,
	})
	if err != nil {
		return err
	}
	log.Info().Msgf("group %s created: %s", group.Name, group.WebURL)

	return nil
}

// GetGroupID
func (gl *GitLabWrapper) GetGroupID(groups []gitlab.Group, groupName string) (int, error) {
	for _, g := range groups {
		if g.Name == groupName {
			return g.ID, nil
		}
	}
	return 0, errors.New(fmt.Sprintf("group %s not found", groupName))
}

// GetGroups
func (gl *GitLabWrapper) GetGroups() ([]gitlab.Group, error) {
	owned := true

	container := make([]gitlab.Group, 0)
	for nextPage := 1; nextPage > 0; {
		groups, resp, err := gl.Client.Groups.ListGroups(&gitlab.ListGroupsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    nextPage,
				PerPage: 10,
			},
			Owned: &owned,
		})
		if err != nil {
			return []gitlab.Group{}, err
		}
		for _, group := range groups {
			container = append(container, *group)
		}
		nextPage = resp.NextPage
	}

	return container, nil
}

// GetProjectID
func (gl *GitLabWrapper) GetProjectID(projectName string) (int, error) {
	owned := true

	container := make([]gitlab.Project, 0)
	for nextPage := 1; nextPage > 0; {
		projects, resp, err := gl.Client.Projects.ListProjects(&gitlab.ListProjectsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    nextPage,
				PerPage: 10,
			},
			Owned: &owned,
		})
		if err != nil {
			return 0, err
		}
		for _, project := range projects {
			container = append(container, *project)
		}
		nextPage = resp.NextPage
	}

	for _, project := range container {
		if project.Name == projectName {
			return project.ID, nil
		}
	}

	return 0, errors.New(fmt.Sprintf("could not get project ID for project %s", projectName))
}

// GetProjects
func (gl *GitLabWrapper) GetProjects() ([]gitlab.Project, error) {
	owned := true

	container := make([]gitlab.Project, 0)
	for nextPage := 1; nextPage > 0; {
		projects, resp, err := gl.Client.Projects.ListProjects(&gitlab.ListProjectsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    nextPage,
				PerPage: 10,
			},
			Owned: &owned,
		})
		if err != nil {
			return []gitlab.Project{}, err
		}
		for _, project := range projects {
			container = append(container, *project)
		}
		nextPage = resp.NextPage
	}

	return container, nil
}

// GetSubGroupID
func (gl *GitLabWrapper) GetSubGroupID(groupID int, subGroupName string) (int, error) {
	owned := true

	container := make([]gitlab.Group, 0)
	for nextPage := 1; nextPage > 0; {
		subgroups, resp, err := gl.Client.Groups.ListSubGroups(groupID, &gitlab.ListSubGroupsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    nextPage,
				PerPage: 10,
			},
			Owned: &owned,
		})
		if err != nil {
			return 0, err
		}
		for _, subgroup := range subgroups {
			container = append(container, *subgroup)
		}
		nextPage = resp.NextPage
	}

	for _, g := range container {
		if g.Name == subGroupName {
			return g.ID, nil
		}
	}

	return 0, errors.New(fmt.Sprintf("subgroup %s not found", subGroupName))
}

// GetSubGroups
func (gl *GitLabWrapper) GetSubGroups(groupID int) ([]gitlab.Group, error) {
	owned := true

	container := make([]gitlab.Group, 0)
	for nextPage := 1; nextPage > 0; {
		subgroups, resp, err := gl.Client.Groups.ListSubGroups(groupID, &gitlab.ListSubGroupsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    nextPage,
				PerPage: 10,
			},
			Owned: &owned,
		})
		if err != nil {
			return []gitlab.Group{}, err
		}
		for _, subgroup := range subgroups {
			container = append(container, *subgroup)
		}
		nextPage = resp.NextPage
	}

	return container, nil
}

// FindProjectInGroup
func (gl *GitLabWrapper) FindProjectInGroup(projects []gitlab.Project, projectName string) (bool, error) {
	for _, pj := range projects {
		if pj.Name == projectName {
			return true, nil
		}
	}

	return false, errors.New(fmt.Sprintf("project %s not found", projectName))
}

// User Management

// AddUserSSHKey
func (gl *GitLabWrapper) AddUserSSHKey(keyTitle string, keyValue string) error {
	_, _, err := gl.Client.Users.AddSSHKey(&gitlab.AddSSHKeyOptions{
		Title: &keyTitle,
		Key:   &keyValue,
	})
	if err != nil {
		return err
	}

	return nil
}

// DeleteUserSSHKey
func (gl *GitLabWrapper) DeleteUserSSHKey(keyTitle string) error {
	allkeys, err := gl.GetUserSSHKeys()
	if err != nil {
		return err
	}

	var keyID int = 0
	for _, key := range allkeys {
		if key.Title == keyTitle {
			keyID = key.ID
		}
	}

	if keyID == 0 {
		return errors.New(fmt.Sprintf("could not find ssh key %s so it will not be deleted - you may need to delete it manually", keyTitle))
	}
	_, err = gl.Client.Users.DeleteSSHKey(keyID)
	if err != nil {
		return err
	}
	log.Info().Msgf("deleted gitlab ssh key %s", keyTitle)

	return nil
}

// GetUserSSHKeys
func (gl *GitLabWrapper) GetUserSSHKeys() ([]*gitlab.SSHKey, error) {
	keys, _, err := gl.Client.Users.ListSSHKeys()
	if err != nil {
		return []*gitlab.SSHKey{}, err
	}

	return keys, nil
}

// Container Registry

// GetProjectContainerRegistryRepositories
func (gl *GitLabWrapper) GetProjectContainerRegistryRepositories(projectName string) ([]gitlab.RegistryRepository, error) {
	projectID, err := gl.GetProjectID(projectName)
	if err != nil {
		return []gitlab.RegistryRepository{}, err
	}

	container := make([]gitlab.RegistryRepository, 0)
	for nextPage := 1; nextPage > 0; {
		repositories, resp, err := gl.Client.ContainerRegistry.ListProjectRegistryRepositories(projectID, &gitlab.ListRegistryRepositoriesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    nextPage,
				PerPage: 10,
			},
		})
		if err != nil {
			return []gitlab.RegistryRepository{}, err
		}
		for _, subgroup := range repositories {
			container = append(container, *subgroup)
		}
		nextPage = resp.NextPage
	}

	return container, nil
}

// DeleteProjectContainerRegistryRepository
func (gl *GitLabWrapper) DeleteContainerRegistryRepository(projectName string, repositoryID int) error {
	projectID, err := gl.GetProjectID(projectName)
	if err != nil {
		return err
	}

	// Delete any tags
	nameRegEx := ".*"
	_, err = gl.Client.ContainerRegistry.DeleteRegistryRepositoryTags(projectID, repositoryID, &gitlab.DeleteRegistryRepositoryTagsOptions{
		NameRegexpDelete: &nameRegEx,
	})
	if err != nil {
		return err
	}
	log.Info().Msgf("removed all tags from container registry for project %s", projectName)

	// Delete repository
	_, err = gl.Client.ContainerRegistry.DeleteRegistryRepository(projectID, repositoryID)
	if err != nil {
		return err
	}
	log.Info().Msgf("deleted container registry for project %s", projectName)

	return nil
}

// Token & Key Management

// CreateProjectDeployToken
func (gl *GitLabWrapper) CreateProjectDeployToken(projectName string, p *DeployTokenCreateParameters) (string, error) {
	projectID, err := gl.GetProjectID(projectName)
	if err != nil {
		return "", err
	}

	// Check to see if the token already exists
	allTokens, err := gl.ListProjectDeployTokens(projectName)
	if err != nil {
		return "", err
	}

	var exists bool = false
	for _, token := range allTokens {
		if token.Name == p.Name {
			exists = true
		}
	}

	if !exists {
		token, _, err := gl.Client.DeployTokens.CreateProjectDeployToken(projectID, &gitlab.CreateProjectDeployTokenOptions{
			Name:     &p.Name,
			Username: &p.Username,
			Scopes:   &p.Scopes,
		})
		if err != nil {
			return "", err
		}
		log.Info().Msgf("created deploy token %s", token.Name)

		return token.Token, nil
	} else {
		log.Info().Msgf("deploy token %s already exists - skipping", p.Name)
		return "", nil
	}
}

// DeleteProjectDeployToken
func (gl *GitLabWrapper) DeleteProjectDeployToken(projectName string, tokenName string) error {
	projectID, err := gl.GetProjectID(projectName)
	if err != nil {
		return err
	}

	allTokens, err := gl.ListProjectDeployTokens(projectName)
	if err != nil {
		return err
	}

	var exists bool = false
	var tokenID int
	for _, token := range allTokens {
		if token.Name == tokenName {
			exists = true
			tokenID = token.ID
		}
	}

	if exists {
		_, err = gl.Client.DeployTokens.DeleteProjectDeployToken(projectID, tokenID)
		if err != nil {
			return err
		}
		log.Info().Msgf("deleted deploy token %s", tokenName)
	}

	return nil

}

// ListProjectDeployTokens
func (gl *GitLabWrapper) ListProjectDeployTokens(projectName string) ([]gitlab.DeployToken, error) {
	projectID, err := gl.GetProjectID(projectName)
	if err != nil {
		return []gitlab.DeployToken{}, err
	}

	container := make([]gitlab.DeployToken, 0)
	for nextPage := 1; nextPage > 0; {
		tokens, resp, err := gl.Client.DeployTokens.ListProjectDeployTokens(projectID, &gitlab.ListProjectDeployTokensOptions{
			Page:    nextPage,
			PerPage: 10,
		})
		if err != nil {
			return []gitlab.DeployToken{}, err
		}
		for _, token := range tokens {
			container = append(container, *token)
		}
		nextPage = resp.NextPage
	}

	return container, nil
}

// DeleteProjectWebhook
func (gl *GitLabWrapper) DeleteProjectWebhook(projectName string, url string) error {
	projectID, err := gl.GetProjectID(projectName)
	if err != nil {
		return err
	}

	webhooks, err := gl.ListProjectWebhooks(projectID)
	if err != nil {
		return err
	}

	var hookID int = 0
	for _, hook := range webhooks {
		if hook.ProjectID == projectID && hook.URL == url {
			hookID = hook.ID
		}
	}
	if hookID == 0 {
		return errors.New(fmt.Sprintf("no webhooks were found for project %s given search parameters", projectName))
	}
	_, err = gl.Client.Projects.DeleteProjectHook(projectID, hookID)
	if err != nil {
		return err
	}
	log.Info().Msgf("deleted hook %s/%s", projectName, url)

	return nil
}

// ListProjectWebhooks returns all webhooks for a project
func (gl *GitLabWrapper) ListProjectWebhooks(projectID int) ([]gitlab.ProjectHook, error) {
	container := make([]gitlab.ProjectHook, 0)
	for nextPage := 1; nextPage > 0; {
		hooks, resp, err := gl.Client.Projects.ListProjectHooks(projectID, &gitlab.ListProjectHooksOptions{
			Page:    nextPage,
			PerPage: 10,
		})
		if err != nil {
			return []gitlab.ProjectHook{}, err
		}
		for _, hook := range hooks {
			container = append(container, *hook)
		}
		nextPage = resp.NextPage
	}
	return container, nil
}
