/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gitlab

import "github.com/xanzy/go-gitlab"

// GitLabWrapper holds gitlab cloud client info and provides and interface
// to its functions
type GitLabWrapper struct {
	Client          *gitlab.Client
	ParentGroupID   int
	ParentGroupPath string
}

// DeployTokenCreateParameters holds values to be passed to a function to create
// deploy tokens
type DeployTokenCreateParameters struct {
	Name     string
	Username string
	Scopes   []string
}
