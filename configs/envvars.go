package configs

import (
	"fmt"
	"log"
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
			log.Printf(errorMsg)
			return fmt.Errorf(errorMsg)
		}
	}

	log.Println("all environment variables are set")

	return nil
}
