package aws_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/pkg"
	"log"
	"os"
	"strings"
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

// this is called after cluster destruction, and will fail if VPC is still active
func TestVPCByTagIntegration(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	clusterName := os.Getenv("CLUSTER_NAME")

	awsConfig, err := aws.NewAws()
	if err != nil {
		t.Error(err)
	}

	ec2Client := ec2.NewFromConfig(awsConfig)

	filterType := "tag:ClusterName"
	vpcData, err := ec2Client.DescribeVpcs(context.Background(), &ec2.DescribeVpcsInput{
		Filters: []ec2Types.Filter{
			{
				Name:   &filterType,
				Values: []string{clusterName},
			},
		},
	})
	if err != nil {
		t.Error(err)
	}

	if len(vpcData.Vpcs) == 0 {
		t.Errorf("there is no VPC for the cluster %q", clusterName)
	}

	for _, v := range vpcData.Vpcs {
		if v.State != "available" {
			t.Errorf("there is a VPC for the %q cluster, but the status is not available", clusterName)
		}
	}
}

// todo: this test will be called when cluster is up AND when cluster is down, we must update the condition,
// based on what we want. This test requires AWS_REGION and CLUSTER_NAME.
func TestLoadBalancerByTagIntegration(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	clusterName := os.Getenv("CLUSTER_NAME")

	awsConfig, err := aws.NewAws()
	if err != nil {
		t.Error(err)
	}

	elb := elasticloadbalancing.NewFromConfig(awsConfig)

	loadBalancers, err := elb.DescribeLoadBalancers(
		context.Background(),
		&elasticloadbalancing.DescribeLoadBalancersInput{},
	)
	if err != nil {
		t.Error(err)
	}

	var regionLoadBalancers []string
	for _, loadBalancerItem := range loadBalancers.LoadBalancerDescriptions {
		regionLoadBalancers = append(regionLoadBalancers, *loadBalancerItem.LoadBalancerName)
	}

	loadBalancersTags, err := elb.DescribeTags(context.Background(), &elasticloadbalancing.DescribeTagsInput{
		LoadBalancerNames: regionLoadBalancers,
	})
	if err != nil {
		t.Error(err)
	}

	if len(loadBalancersTags.TagDescriptions) == 0 {
		t.Error(err)
	}

	loadBalancerIsLive := false
	for _, tagDescription := range loadBalancersTags.TagDescriptions {
		for _, b := range tagDescription.Tags {
			if strings.Contains(*b.Key, clusterName) {
				loadBalancerIsLive = true
				break
			}
		}
	}

	if !loadBalancerIsLive {
		t.Errorf("unable to find a load balancer tagged with cluster name %q", clusterName)
	}
}

func TestKMSKeyAliasIntegration(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	clusterName := os.Getenv("CLUSTER_NAME")
	awsRegion := os.Getenv("AWS_REGION")

	if len(clusterName) == 0 || len(awsRegion) == 0 {
		t.Error("environment variables CLUSTER_NAME and AWS_REGION must be informed")
		return
	}

	awsConfig, err := aws.NewAws()
	if err != nil {
		t.Error(err)
	}
	kmsClient := kms.NewFromConfig(awsConfig)

	keyList, err := kmsClient.ListAliases(context.Background(), &kms.ListAliasesInput{})
	if err != nil {
		t.Error(err)
	}

	var activeCKMS string
	for _, ckms := range keyList.Aliases {
		if strings.HasSuffix(*ckms.AliasName, clusterName) {
			activeCKMS = *ckms.TargetKeyId
		}
	}

	if len(activeCKMS) == 0 {
		t.Errorf("unable to find CMKS for the cluster %q", clusterName)
	}

	ckmsKey, err := kmsClient.DescribeKey(context.Background(), &kms.DescribeKeyInput{
		KeyId: &activeCKMS,
	})

	if !ckmsKey.KeyMetadata.Enabled {
		t.Error("wanted CKMS to be enabled, but got it disabled")
	}

	if !strings.Contains(*ckmsKey.KeyMetadata.Arn, awsRegion) {
		t.Errorf("unable to find CKMS at the desired region (%s) for the cluster %q", awsRegion, clusterName)
	}
}

func TestEKSIntegration(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	clusterName := os.Getenv("CLUSTER_NAME")
	awsRegion := os.Getenv("AWS_REGION")

	if len(clusterName) == 0 || len(awsRegion) == 0 {
		t.Error("environment variables CLUSTER_NAME and AWS_REGION must be informed")
		return
	}

	awsConfig, err := aws.NewAws()
	if err != nil {
		t.Error(err)
	}

	eksClient := eks.NewFromConfig(awsConfig)

	eksData, err := eksClient.DescribeCluster(context.Background(), &eks.DescribeClusterInput{
		Name: &clusterName,
	})
	if err != nil {
		t.Error(err)
		return
	}

	if *eksData.Cluster.Name != clusterName {
		t.Errorf("unable to find cluster with cluster name %q", clusterName)
	}
}

func TestEC2Volumes(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	awsRegion := os.Getenv("AWS_REGION")
	if len(awsRegion) == 0 {
		t.Error("environment variables AWS_REGION must be informed")
		return
	}

	awsConfig, err := aws.NewAws()
	if err != nil {
		t.Error(err)
	}

	ec2Client := ec2.NewFromConfig(awsConfig)

	ec2Volumes, err := ec2Client.DescribeVolumes(context.Background(), &ec2.DescribeVolumesInput{})
	if err != nil {
		t.Error(err)
	}

	isVolumeActive := false
	for _, volume := range ec2Volumes.Volumes {
		for _, tag := range volume.Tags {
			if *tag.Value == "owned" &&
				strings.HasSuffix(*tag.Key, "joao_kubefirst_tech") &&
				volume.State == "available" &&
				strings.Contains(*volume.AvailabilityZone, awsRegion) {

				isVolumeActive = true
			}
		}
	}

	if !isVolumeActive {
		t.Error("it should have at least one active volume for the current installation, but got none")
	}
}
