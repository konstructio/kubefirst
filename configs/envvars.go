package configs

import (
	"fmt"
	"os"
)

// CheckEnvironment validate if the required environment variable values are set.
func CheckEnvironment() error {

	requiredEnvValues := map[string]string{
		"AWS_PROFILE": os.Getenv("AWS_PROFILE"),
		"AWS_REGION":  os.Getenv("AWS_REGION"),
	}

	for k, v := range requiredEnvValues {
		if v == "" {
			return fmt.Errorf("%s is not set", k)
		}
	}

	return nil
}