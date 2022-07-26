package configs

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
)

// CheckKubefirstConfigFile validate if ~/.kubefirst file is ready to be consumed.
func CheckKubefirstConfigFile(config *Config) error {
	kubefirstFile := fmt.Sprintf("%s", config.KubefirstConfigFilePath)
	if _, err := os.Stat(kubefirstFile); err != nil {
		errorMsg := fmt.Sprintf("unable to load %q file, error is: %s", config.KubefirstConfigFilePath, err)
		log.Err(err).Msg("")

		return fmt.Errorf(errorMsg)
	}

	log.Info().Msgf("%q file is set", config.KubefirstConfigFilePath)
	return nil
}
