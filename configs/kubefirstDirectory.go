package configs

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
)

// CheckKubefirstDir validate if ~/.k1 directory is ready to be used
func CheckKubefirstDir(config *Config) error {
	k1sDir := fmt.Sprintf("%s", config.K1FolderPath)
	if _, err := os.Stat(k1sDir); err != nil {
		errorMsg := fmt.Sprintf("unable to load \".k1\" directory, error is: %s", err)
		log.Error().Msg(errorMsg)
		return fmt.Errorf(errorMsg)
	}

	log.Info().Msgf("\".k1\" directory found: %s", k1sDir)
	return nil
}
