package configs

import (
	"fmt"
	"log"
	"os"
)

// CheckFlareFile validate if ~/.flare file is ready to be consumed.
func CheckFlareFile(home string) error {
	flareFile := fmt.Sprintf("%s/.flare", home)
	if _, err := os.Stat(flareFile); err != nil {
		errorMsg := fmt.Sprintf("unable to load \".flare\" file, error is: %s", err)
		log.Println(errorMsg)
		return fmt.Errorf(errorMsg)
	}

	log.Println(".flare file is set")
	return nil
}