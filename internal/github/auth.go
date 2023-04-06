/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package github

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
)

const (
	githubApiUrl = "https://api.github.com"
)

var (
	requiredScopes = []string{
		"admin:org",
		"admin:public_key",
		"admin:repo_hook",
		"delete_repo",
		"repo",
		"user",
		"workflow",
		"write:packages",
	}
)

// VerifyTokenPermissions compares scope of the provided token to the required
// scopes for kubefirst functionality
func VerifyTokenPermissions(githubToken string) error {
	req, err := http.NewRequest(http.MethodGet, githubApiUrl, nil)
	if err != nil {
		log.Info().Msg("error setting github owner permissions request")
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", githubToken))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"something went wrong calling GitHub API, http status code is: %d, and response is: %q",
			res.StatusCode,
			string(body),
		)
	}

	// Get token scopes
	scopeHeader := res.Header.Get("X-OAuth-Scopes")
	scopes := make([]string, 0)
	for _, s := range strings.Split(scopeHeader, ",") {
		scopes = append(scopes, strings.TrimSpace(s))
	}

	// Compare token scopes to required scopes
	missingScopes := make([]string, 0)
	for _, ts := range requiredScopes {
		if !pkg.FindStringInSlice(scopes, ts) {
			missingScopes = append(missingScopes, ts)
		}
	}

	// Report on any missing scopes
	if len(missingScopes) != 0 {
		return fmt.Errorf("the supplied github token is missing authorization scopes - please add: %v", missingScopes)
	}

	return nil
}
