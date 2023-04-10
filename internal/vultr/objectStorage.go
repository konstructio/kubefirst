/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vultr

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
	"github.com/vultr/govultr/v3"
)

// CreateObjectStorage creates a Vultr object storage resource
func (c *VultrConfiguration) CreateObjectStorage(region string, storeName string) (govultr.ObjectStorage, error) {
	// Get cluster id of object storage cluster for region
	clusters, _, _, err := c.Client.ObjectStorage.ListCluster(c.Context, &govultr.ListOptions{
		Region: region,
	})
	if err != nil {
		return govultr.ObjectStorage{}, err
	}
	var clid int = 0
	for _, cluster := range clusters {
		if cluster.Region == region {
			clid = cluster.ID
		}
	}
	if clid == 0 {
		return govultr.ObjectStorage{}, fmt.Errorf("could not find object storage cluster for region %s", region)
	}

	objst, _, err := c.Client.ObjectStorage.Create(c.Context, clid, storeName)
	if err != nil {
		return govultr.ObjectStorage{}, err
	}

	log.Info().Msgf("waiting for vultr object storage %s to be ready", storeName)
	for i := 0; i < 60; i++ {
		obj, _, err := c.Client.ObjectStorage.Get(c.Context, objst.ID)
		if err != nil {
			return govultr.ObjectStorage{}, err
		}
		switch {
		case obj.Status == "active":
			log.Info().Msgf("vultr object storage %s ready", storeName)
			return *obj, nil
		case i == 120:
			return govultr.ObjectStorage{}, fmt.Errorf("vultr object storage %s is not active", storeName)
		}
		time.Sleep(time.Second * 1)
	}

	return govultr.ObjectStorage{}, err
}

// GetObjectStorage retrieves all Vultr object storage resources
func (c *VultrConfiguration) GetObjectStorage(region string) ([]govultr.ObjectStorage, error) {
	objst, _, _, err := c.Client.ObjectStorage.List(c.Context, &govultr.ListOptions{
		Region: region,
	})
	if err != nil {
		return []govultr.ObjectStorage{}, err
	}

	return objst, nil
}

// CreateObjectStorageBucket leverages minio to create a bucket within Vultr object storage
func (c *VultrConfiguration) CreateObjectStorageBucket(cr VultrBucketCredentials, bucketName string) error {
	ctx := context.Background()
	useSSL := true

	// Initialize minio client object.
	minioClient, err := minio.New(cr.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cr.AccessKey, cr.SecretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return fmt.Errorf("error initializing minio client for vultr: %s", err)
	}

	location := "us-east-1"
	err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		return fmt.Errorf("error creating bucket %s for %s: %s", bucketName, cr.Endpoint, err)
	}

	return nil
}
