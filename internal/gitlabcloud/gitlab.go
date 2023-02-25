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
	log.Info().Msgf("subgroup %s added to group %s", subGroupID, group.Name)
	return nil
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

// Users

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

// GetUserSSHKeys
func (gl *GitLabWrapper) GetUserSSHKeys() ([]*gitlab.SSHKey, error) {
	keys, _, err := gl.Client.Users.ListSSHKeys()
	if err != nil {
		return []*gitlab.SSHKey{}, err
	}
	return keys, nil
}
