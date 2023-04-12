/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package digitalocean

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// CreateSpaceBucket
func (c *DigitaloceanConfiguration) CreateSpaceBucket(cr DigitaloceanSpacesCredentials, bucketName string) error {
	ctx := context.Background()
	useSSL := true

	// Initialize minio client object.
	minioClient, err := minio.New(cr.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cr.AccessKey, cr.SecretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return fmt.Errorf("error initializing minio client for digitalocean: %s", err)
	}

	location := "us-east-1"
	err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		return fmt.Errorf("error creating bucket %s for %s: %s", bucketName, cr.Endpoint, err)
	}

	spaces, err := minioClient.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("could not list spaces: %s", err)
	}
	for _, space := range spaces {
		fmt.Println(space.Name)
	}

	return nil
}
