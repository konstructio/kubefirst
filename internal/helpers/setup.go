/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package helpers

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// DisplayLogHints prints info to the terminal regarding log streaming
func DisplayLogHints() {
	logFile := viper.GetString("k1-paths.log-file")

	fmt.Println(strings.Repeat("-", 48))
	fmt.Printf("Follow your logs in a new terminal with: \n   tail -f -n +1 %s \n", logFile)
	fmt.Println(strings.Repeat("-", 48))
}

// GetCompletionFlags gets specific config flags to mark status of an install
func GetCompletionFlags() CompletionFlags {
	return CompletionFlags{
		CloudProvider: viper.GetString("kubefirst.cloud-provider"),
		GitProvider:   viper.GetString("kubefirst.git-provider"),
		SetupComplete: viper.GetBool("kubefirst.setup-complete"),
	}
}

// SetCompletionFlags sets specific config flags to mark status of an install
func SetCompletionFlags(cloudProvider string, gitProvider string) {
	viper.Set("kubefirst.cloud-provider", cloudProvider)
	viper.Set("kubefirst.git-provider", gitProvider)
	viper.Set("kubefirst.setup-complete", true)
	viper.WriteConfig()
}
