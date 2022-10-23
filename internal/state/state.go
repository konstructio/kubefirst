package state

import (
	"fmt"
	"log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/spf13/viper"
)

// UploadKubefirstToStateStore - Send kubefirst file to state store
func UploadKubefirstToStateStore(dryRun bool) error {
	if dryRun {
		log.Printf("[#99] Dry-run mode, UploadKubefirstToStateStore skipped.")
		return nil
	}
	config := configs.ReadConfig()
	// upload kubefirst config to user state S3 bucket
	stateStoreBucket := viper.GetString("bucket.state-store.name")
	err := aws.UploadFile(stateStoreBucket, config.KubefirstConfigFileName, config.KubefirstConfigFilePath)
	if err != nil {
		return fmt.Errorf("unable to upload Kubefirst cofiguration file to the S3 bucket, error is: %v", err)
	}
	log.Printf("Kubefirst configuration file was upload to AWS S3 at %q bucket name", stateStoreBucket)

	return nil
}
