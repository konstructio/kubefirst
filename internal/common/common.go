/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package common

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/kubefirst/runtime/configs"
	"github.com/tcnksm/go-latest"
)

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
func versionCheck() (res *latest.CheckResponse, skip bool) {
	githubTag := &latest.GithubTag{
		Owner:             "kubefirst",
		Repository:        "kubefirst",
		FixVersionStrFunc: latest.DeleteFrontV(),
	}
	res, err := latest.Check(githubTag, strings.Replace(configs.K1Version, "v", "", 1))
	if err != nil {
		fmt.Printf("checking for a newer version failed with: %s", err)
		return nil, true
	}

	return res, false
}
