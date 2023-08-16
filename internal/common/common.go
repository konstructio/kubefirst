/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package common

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"runtime"
	"strings"

	"github.com/kubefirst/kubefirst/configs"
)

type CheckResponse struct {
	// Current is current latest version on source.
	Current string

	// Outdate is true when target version is less than Curernt on source.
	Outdated bool

	// Latest is true when target version is equal to Current on source.
	Latest bool

	// New is true when target version is greater than Current on source.
	New bool
}

// CheckForVersionUpdate determines whether or not there is a new cli version available
func CheckForVersionUpdate() {
	if configs.K1Version != configs.DefaultK1Version {
		res, skip := versionCheck()
		if !skip {
			if res.Outdated {
				switch runtime.GOOS {
				case "darwin":
					fmt.Printf("A newer version (v%s) is available! Please upgrade with: \"brew update && brew upgrade kubefirst\"\n", res.Current)
				default:
					fmt.Printf("A newer version (v%s) is available! \"https://github.com/kubefirst/kubefirst/blob/main/build/README.md\"\n", res.Current)
				}
			}
		}
	}
}

// versionCheck compares local to remote version
func versionCheck() (res *CheckResponse, skip bool) {
	var latestVersion string

	resp, err := http.Get("https://raw.githubusercontent.com/Homebrew/homebrew-core/master/Formula/k/kubefirst.rb")

	if err != nil {
		fmt.Printf("checking for a newer version failed with: %s", err)
		return nil, true
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)

		if err != nil {
			fmt.Printf("checking for a newer version failed with: %s", err)
			return nil, true
		}

		bodyString := string(bodyBytes)
		for _, sentence := range strings.Split(bodyString, "\n") {
			if strings.Contains(sentence, "url \"https://github.com/kubefirst/kubefirst/archive/refs/tags/") {
				re := regexp.MustCompile(`.*/v(.*).tar.gz"`)
				matches := re.FindStringSubmatch(sentence)
				latestVersion = matches[1]
			}
		}
	} else {
		fmt.Printf("checking for a newer version failed with: %s", err)
		return nil, true
	}

	return &CheckResponse{
		Current:  configs.K1Version,
		Outdated: latestVersion < configs.K1Version,
		Latest:   latestVersion == configs.K1Version,
		New:      configs.K1Version > latestVersion,
	}, false
}
