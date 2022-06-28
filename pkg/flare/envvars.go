package flare

import (
	"log"
	"os"
	)

//Verify the state of the ".flare" file used to config provisioning.
//
// Output:
//   $PATH/.flare
func CheckEnvironment() bool {

	if value := os.Getenv("AWS_REGION"); value == "" {
		log.Printf("AWS_REGION env var not set.")	
		log.Printf("AWS_REGION is recommended for execution.")		  	  
	} else {
		log.Printf("AWS_REGION env var set: %s",value)		  
	}
	if value := os.Getenv("AWS_PROFILE"); value == "" {
		log.Printf("AWS_PROFILE env var not set.")
		log.Printf("AWS_PROFILE is recommended for execution.")		  
	} else {
		log.Printf("AWS_PROFILE env var set: %s",value)		  
	}



	return true	
}