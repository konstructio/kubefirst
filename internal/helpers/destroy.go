/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package helpers

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// EvalDestroy determines whether or not there are active kubefirst platforms
// If there are not, an error is returned
func EvalDestroy(expectedCloudProvider string, expectedGitProvider string) (bool, error) {
	cloudProvider := viper.GetString("kubefirst.cloud-provider")
	gitProvider := viper.GetString("kubefirst.git-provider")
	setupComplete := viper.GetBool("kubefirst.setup-complete")

	if !setupComplete {
		return false, fmt.Errorf(
			"there are no active kubefirst platforms to destroy.\n\tTo get started, run: kubefirst %s create -h\n",
			expectedCloudProvider,
		)
	}

	if cloudProvider == "" || gitProvider == "" {
		return false, fmt.Errorf("could not parse cloud and git provider information from config")
	}
	log.Info().Msgf("Verified %s platform using %s - continuing with destroy...", expectedCloudProvider, expectedGitProvider)

	return true, nil
}
