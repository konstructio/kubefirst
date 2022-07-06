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
		return fmt.Errorf("unable to load \".kubefirst\" directory, error is: %s", err)
	}

	log.Printf("\".kubefirst\" file found: %s", k1sDir)
	return nil
}
