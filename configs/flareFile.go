package configs

import (
	"fmt"
	"os"
)

// CheckFlareFile validate if ~/.flare file is ready to be consumed.
func CheckFlareFile(home string) error {
	flareFile := fmt.Sprintf("%s/.flare", home)
	if _, err := os.Stat(flareFile); err != nil {
		return fmt.Errorf("unable to load \".flare\" file, error is: %s", err)
	}
	return nil
}