package configs

import (
	"fmt"
	"log"
	"os"
)

// CheckKubefirstDir validate if ~/.kubefirst directory is ready to be used
func CheckKubefirstDir(home string) error {
	k1sDir := fmt.Sprintf("%s/.kubefirst", home)
	if _, err := os.Stat(k1sDir); err != nil {
		errorMsg := fmt.Sprintf("unable to load \".kubefirst\" directory, error is: %s", err)
		log.Println(errorMsg)
		return fmt.Errorf(errorMsg)
	}

	log.Printf("\".kubefirst\" directory found: %s", k1sDir)
	return nil
}
