package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func (conf *AWSConfiguration) CreateBucket(bucketName string) (*s3.CreateBucketOutput, error) {

	s3Client := s3.NewFromConfig(conf.Config)

	s3CreateBucketInput := &s3.CreateBucketInput{}
	s3CreateBucketInput.Bucket = aws.String(bucketName)

	if conf.Config.Region != RegionUsEast1 {
		s3CreateBucketInput.CreateBucketConfiguration = &s3Types.CreateBucketConfiguration{
			LocationConstraint: s3Types.BucketLocationConstraint(conf.Config.Region),
		}
	}

	bucket, err := s3Client.CreateBucket(context.Background(), s3CreateBucketInput)
	if err != nil {
		return &s3.CreateBucketOutput{}, err
	}

	versionConfigInput := &s3.PutBucketVersioningInput{
		Bucket: aws.String(bucketName),
		VersioningConfiguration: &s3Types.VersioningConfiguration{
			Status: s3Types.BucketVersioningStatusEnabled,
		},
	}

	_, err = s3Client.PutBucketVersioning(context.Background(), versionConfigInput)
	if err != nil {
		return &s3.CreateBucketOutput{}, err
	}
	return bucket, nil
}

func (conf *AWSConfiguration) ListBuckets() (*s3.ListBucketsOutput, error) {
	fmt.Println("listing buckets")
	s3Client := s3.NewFromConfig(conf.Config)

	buckets, err := s3Client.ListBuckets(context.Background(), &s3.ListBucketsInput{})
	if err != nil {
		return &s3.ListBucketsOutput{}, err
	}

	return buckets, nil
}
