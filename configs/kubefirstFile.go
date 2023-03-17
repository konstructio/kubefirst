package configs

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

// CheckKubefirstConfigFile validate if ~/.kubefirst file is ready to be consumed.
func CheckKubefirstConfigFile(config *Config) error {
	if _, err := os.Stat(config.KubefirstConfigFilePath); err != nil {
		errorMsg := fmt.Sprintf("unable to load %q file, error is: %s", config.KubefirstConfigFilePath, err)
		log.Error().Msg(errorMsg)
		return fmt.Errorf(errorMsg)
	}
	log.Info().Msgf("%q file is set", config.KubefirstConfigFilePath)
	return nil
}
