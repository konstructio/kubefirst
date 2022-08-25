package aws_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/pkg"
	"log"
	"testing"
)

// TestAreS3BucketsLiveIntegration checks if bucket exists
func TestAreS3BucketsLiveIntegration(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	config := configs.ReadConfig()
	err := pkg.SetupViper(config)
	if err != nil {
		t.Error(err)
	}

	currentInstallationBuckets := aws.ListBucketsInUse()

	awsConfig, err := aws.NewAws()
	if err != nil {
		t.Errorf("unable to connect to AWS, error is: %s", err)
	}

	s3client := s3.NewFromConfig(awsConfig)

	for _, bucketName := range currentInstallationBuckets {

		fmt.Println(bucketName)
		_, err = s3client.HeadBucket(context.Background(), &s3.HeadBucketInput{
			Bucket: &bucketName,
		})

		var s3NotFound *s3Types.NotFound
		if errors.As(err, &s3NotFound) {
			log.Printf("bucket %s don't exist", bucketName)
			t.Error(err)
		}
		if err != nil {
			t.Error(err)
		}
	}
}

// TestAreS3BucketsDestroyedIntegration check if desired S3 buckets are deleted, if the bucket exist, the test fails
func TestAreS3BucketsDestroyedIntegration(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	config := configs.ReadConfig()
	err := pkg.SetupViper(config)
	if err != nil {
		t.Error(err)
	}

	currentInstallationBuckets := aws.ListBucketsInUse()

	awsConfig, err := aws.NewAws()
	if err != nil {
		t.Errorf("unable to connect to AWS, error is: %s", err)
	}

	s3client := s3.NewFromConfig(awsConfig)

	for _, bucketName := range currentInstallationBuckets {

		_, err = s3client.HeadBucket(context.Background(), &s3.HeadBucketInput{
			Bucket: &bucketName,
		})
		if err == nil {
			t.Error(err)
		}
	}
}
