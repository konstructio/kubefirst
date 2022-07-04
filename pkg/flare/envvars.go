package flare

import (
	"log"
	"fmt"
	"os"
	)

//Verify the state of the ".flare" file used to config provisioning.
//
// Output:
//   $PATH/.flare
func CheckEnvironment(printOut bool) bool {

	if value := os.Getenv("AWS_REGION"); value == "" {
		log.Printf("AWS_REGION env var not set.")	
		log.Printf("AWS_REGION is recommended for execution.")
		if printOut {
			fmt.Printf("AWS_REGION env var not set.\n")	
			fmt.Printf("AWS_REGION is recommended for execution.\n")
		}		  	  
	} else {
		log.Printf("AWS_REGION env var set: %s",value)	
		if printOut {
			fmt.Printf("AWS_REGION env var set: %s\n",value)	
		}	  
	}

	if value := os.Getenv("AWS_PROFILE"); value == "" {
		log.Printf("AWS_PROFILE env var not set.")
		log.Printf("AWS_PROFILE is recommended for execution.")	
		if printOut {
			log.Printf("AWS_PROFILE env var not set. \n")
			log.Printf("AWS_PROFILE is recommended for execution.\n")	
		}	  
	} else {
		log.Printf("AWS_PROFILE env var set: %s",value)		  
		if printOut {
			log.Printf("AWS_PROFILE env var set: %s\n",value)
		}
	}



	return true	
}