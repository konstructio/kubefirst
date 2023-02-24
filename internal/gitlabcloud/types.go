package gitlabcloud

import "github.com/xanzy/go-gitlab"

// GitLabWrapper holds gitlab cloud client info and provides and interface
// to its functions
type GitLabWrapper struct {
	Client *gitlab.Client
}
