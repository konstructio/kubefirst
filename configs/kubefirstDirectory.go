package configs

import (
	"fmt"
	"log"
	"os"
)

// CheckKubefirstDir validate if ~/.k1srt directory is ready to be used
func CheckKubefirstDir(config *Config) error {
	k1sDir := fmt.Sprintf("%s", config.K1srtFolderPath)
	if _, err := os.Stat(k1sDir); err != nil {
		errorMsg := fmt.Sprintf("unable to load \".k1srt\" directory, error is: %s", err)
		log.Println(errorMsg)
		return fmt.Errorf(errorMsg)
	}

	log.Printf("\".k1srt\" directory found: %s", k1sDir)
	return nil
}
