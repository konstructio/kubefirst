package flare

import (
	"errors"
	"log"
	"os"
)

// CheckEnvironment checks if environment variables are set.
func CheckEnvironment() bool {

	if value := os.Getenv("AWS_REGION"); value == "" {
		log.Panic(errors.New("AWS_REGION is not set"))
	}

	if value := os.Getenv("AWS_PROFILE"); value == "" {
		log.Panic(errors.New("AWS_PROFILE is not set"))
	}

	log.Printf("AWS_REGION value: %s", os.Getenv("AWS_REGION"))
	log.Printf("AWS_PROFILE value: %s", os.Getenv("AWS_PROFILE"))

	return true
}
