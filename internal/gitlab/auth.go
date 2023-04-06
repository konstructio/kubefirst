/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gitlab

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
)

const (
	gitlabApiUrl = "https://gitlab.com/api/v4"
)

var (
	requiredScopes = []string{
		"read_api",
		"read_user",
		"read_repository",
		"write_repository",
		"read_registry",
		"write_registry",
	}
)

// VerifyTokenPermissions compares scope of the provided token to the required
// scopes for kubefirst functionality
func VerifyTokenPermissions(gitlabToken string) error {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/personal_access_tokens/self", gitlabApiUrl), nil)
	if err != nil {
		log.Info().Msg("error setting gitlab owner permissions request")
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gitlabToken))

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
			"something went wrong calling GitLab API, http status code is: %d, and response is: %q",
			res.StatusCode,
			string(body),
		)
	}

	// Get token scopes
	var responseJson interface{}
	err = json.Unmarshal(body, &responseJson)
	if err != nil {
		return err
	}
	responseJsonMap := responseJson.(map[string]interface{})
	scopes := responseJsonMap["scopes"].([]interface{})
	scopesSlice := make([]string, 0)
	for _, s := range scopes {
		scopesSlice = append(scopesSlice, string(s.(string)))
	}

	// api allows all access so we won't need to check the rest
	if pkg.FindStringInSlice(scopesSlice, "api") {
		return nil
	}

	// Compare token scopes to required scopes
	missingScopes := make([]string, 0)
	for _, ts := range requiredScopes {
		if !pkg.FindStringInSlice(scopesSlice, ts) {
			missingScopes = append(missingScopes, ts)
		}
	}

	// Report on any missing scopes
	if !pkg.FindStringInSlice(scopesSlice, "api") && len(missingScopes) != 0 {
		return fmt.Errorf("the supplied github token is missing authorization scopes - please add: %v", missingScopes)
	}

	return nil
}
