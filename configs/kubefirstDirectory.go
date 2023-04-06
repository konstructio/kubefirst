/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package configs

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

// CheckKubefirstDir validate if ~/.k1 directory is ready to be used
func CheckKubefirstDir(config *Config) error {
	if _, err := os.Stat(config.K1FolderPath); err != nil {
		return fmt.Errorf("unable to load \".k1\" directory, error is: %s", err)
	}

	log.Info().Msgf("\".k1\" directory found: %s", config.K1FolderPath)
	return nil
}
