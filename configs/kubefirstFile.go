package configs

import (
	"fmt"
	"log"
	"os"
)

// CheckKubefirstConfigFile validate if ~/.kubefirst file is ready to be consumed.
func CheckKubefirstConfigFile(config *Config) error {
	kubefirstFile := fmt.Sprintf("%s", config.KubefirstConfigFilePath)
	if _, err := os.Stat(kubefirstFile); err != nil {
		errorMsg := fmt.Sprintf("unable to load %q file, error is: %s", config.KubefirstConfigFilePath, err)
		log.Println(errorMsg)
		return fmt.Errorf(errorMsg)
	}

	log.Printf("%q file is set", config.KubefirstConfigFilePath)
	return nil
}
