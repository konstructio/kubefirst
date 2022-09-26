package ciTools

import (
	"fmt"

	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/spf13/viper"
)

func CreateBucket() (string, error) {
	randomName := viper.GetString("bucket.rand")
	bucket := "ci-state"
	bucketName := fmt.Sprintf("k1-%s-%s", bucket, randomName)
	aws.CreateBucket(false, bucketName)

	viper.Set(fmt.Sprintf("bucket.%s.created", bucket), true)
	viper.Set(fmt.Sprintf("bucket.%s.name", bucket), bucketName)
	viper.WriteConfig()

	return bucketName, nil
}

func DestroyBucket() error {
	randomName := viper.GetString("bucket.rand")
	bucket := "ci-state"
	bucketName := fmt.Sprintf("k1-%s-%s", bucket, randomName)
	bucketRegion := viper.GetString("aws.region")
	aws.DestroyBucketObjectsAndVersions(bucketName, bucketRegion)

	viper.Set(fmt.Sprintf("bucket.%s.destroyed", bucket), true)
	viper.Set(fmt.Sprintf("bucket.%s.name", bucket), bucketName)
	viper.WriteConfig()

	return nil
}
