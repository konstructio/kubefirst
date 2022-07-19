package configs

import (
	"fmt"
	"log"
	"os"
)

// CheckKubefirstDir validate if ~/.k1 directory is ready to be used
func CheckKubefirstDir(config *Config) error {
	k1sDir := fmt.Sprintf("%s", config.K1FolderPath)
	if _, err := os.Stat(k1sDir); err != nil {
		errorMsg := fmt.Sprintf("unable to load \".k1\" directory, error is: %s", err)
		log.Println(errorMsg)
		return fmt.Errorf(errorMsg)
	}

	log.Printf("\".k1\" directory found: %s", k1sDir)
	return nil
}
