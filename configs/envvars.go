package configs

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
)

// CheckEnvironment validate if the required environment variable values are set.
func CheckEnvironment() error {

	requiredEnvValues := map[string]string{
		"AWS_PROFILE": os.Getenv("AWS_PROFILE"),
	}

	for k, v := range requiredEnvValues {
		if v == "" {
			errorMsg := fmt.Sprintf("%s is not set", k)
			log.Err(errors.New(errorMsg)).Send()
			return fmt.Errorf(errorMsg)
		}
	}

	log.Info().Msg("all environment variables are set")

	return nil
}
