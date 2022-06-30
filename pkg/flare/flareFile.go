package flare

import (
	"log"
	"fmt"
	"os"
	"errors"
	)

//Verify the state of the ".flare" file used to config provisioning.
//
// Output:
//   $PATH/.flare
func CheckFlareFile(home string, printOut bool) string {
	flareFile :=  fmt.Sprintf("%s/.flare", home)
	if _, err := os.Stat(flareFile); err == nil {
		// path/to/whatever exists
		log.Printf("\".flare\" file found: %s", flareFile)
		if printOut {
			fmt.Printf("\".flare\" file found: %s \n", flareFile)
		}
	} else if errors.Is(err, os.ErrNotExist) {
		// path/to/whatever does *not* exist
		log.Printf("\".flare\" file not found: %s", flareFile)
		log.Printf("	\".flare\" is needed to guide installation process" )
		if printOut {
			fmt.Printf("\".flare\" file not found: %s\n", flareFile)
			fmt.Printf("	\".flare\" is needed to guide installation process\n" )
		}		  
	} else {
		// Schrodinger: file may or may not exist. See err for details.
		// Therefore, do *NOT* use !os.IsNotExist(err) to test for file existence
		log.Printf("Unable to check is \".flare\" if file exists" )		  
		if printOut {
			fmt.Printf("Unable to check is \".flare\" if file exists\n" )
		}
	}
	return flareFile	
}