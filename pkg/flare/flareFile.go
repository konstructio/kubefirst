package flare

import (
	"errors"
	"fmt"
	"log"
	"os"
)

// CheckFlareFile checks if .flare file exists.
func CheckFlareFile(home string) string {
	flareFile := fmt.Sprintf("%s/.flare", home)
	if _, err := os.Stat(flareFile); err == nil {
		// path/to/whatever exists
		log.Printf("\".flare\" file found: %s", flareFile)
	} else if errors.Is(err, os.ErrNotExist) {
		// path/to/whatever does *not* exist
		log.Printf("\".flare\" file not found: %s", flareFile)
		log.Printf("	\".flare\" is needed to guide installation process")
	} else {
		// Schrodinger: file may or may not exist. See err for details.
		// Therefore, do *NOT* use !os.IsNotExist(err) to test for file existence
		log.Printf("Unable to check is \".flare\" if file exists")
	}
	return flareFile
}
