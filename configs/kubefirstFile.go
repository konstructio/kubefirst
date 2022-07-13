package configs

import (
	"fmt"
	"log"
	"os"
)

// CheckKubefirstConfigFile validate if ~/.flare file is ready to be consumed.
func CheckKubefirstConfigFile(config *Config) error {
	flareFile := fmt.Sprintf("%s", config.KubefirstConfigFilePath)
	if _, err := os.Stat(flareFile); err != nil {
		errorMsg := fmt.Sprintf("unable to load %q file, error is: %s", config.KubefirstConfigFilePath, err)
		log.Println(errorMsg)
		return fmt.Errorf(errorMsg)
	}

	log.Printf("%q file is set", config.KubefirstConfigFilePath)
	return nil
}
