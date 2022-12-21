package aws_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	eksTypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
)

// TestAreS3BucketsLiveIntegration checks if bucket exists
func TestAreS3BucketsLiveIntegration(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// this is necessary to load the viper file
	config := configs.ReadConfig()
	err := pkg.SetupViper(config)
	if err != nil {
		t.Error(err)
	}

	currentInstallationBuckets := aws.ListBucketsInUse()

	if len(currentInstallationBuckets) == 0 {
		t.Error("there are no available buckets to be validated")
	}

	awsConfig, err := aws.NewAws()
	if err != nil {
		t.Errorf("unable to connect to AWS, error is: %s", err)
	}

	s3client := s3.NewFromConfig(awsConfig)

	for _, bucketName := range currentInstallationBuckets {
		_, err = s3client.HeadBucket(context.Background(), &s3.HeadBucketInput{
			Bucket: &bucketName,
		})

		var s3NotFound *s3Types.NotFound
		if errors.As(err, &s3NotFound) {
			log.Warn().Msgf("bucket %s don't exist", bucketName)
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
func TestIsVPCByTagDestroyedIntegration(t *testing.T) {

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

	if len(vpcData.Vpcs) > 0 {
		t.Errorf("there is no VPC for the cluster %q", clusterName)
	}

	for _, v := range vpcData.Vpcs {
		if v.State == "available" {
			t.Errorf("there is a VPC for the %q cluster, but the status is not available", clusterName)
		}
	}
}

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

// based on what we want. This test requires AWS_REGION and CLUSTER_NAME.
func TestIsLoadBalancerByTagDestroyedIntegration(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test")
	}

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

	if len(loadBalancers.LoadBalancerDescriptions) > 0 {
		t.Error("wanted no active load balancers")
	}

}

func TestIsKMSKeyAliasDestroyedIntegration(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	config := configs.ReadConfig()

	if len(config.ClusterName) == 0 || len(config.AwsRegion) == 0 {
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
		if strings.HasSuffix(*ckms.AliasName, config.ClusterName) {
			activeCKMS = *ckms.TargetKeyId
		}
	}

	if len(activeCKMS) > 0 {
		t.Errorf("there is at least one active CMKS for the cluster %q", config.ClusterName)
	}
}

func TestIsEKSDestroyedIntegration(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	config := configs.ReadConfig()

	if len(config.ClusterName) == 0 || len(config.AwsRegion) == 0 {
		t.Error("environment variables CLUSTER_NAME and AWS_REGION must be informed")
		return
	}

	awsConfig, err := aws.NewAws()
	if err != nil {
		t.Error(err)
	}

	eksClient := eks.NewFromConfig(awsConfig)

	_, err = eksClient.DescribeCluster(context.Background(), &eks.DescribeClusterInput{
		Name: &config.ClusterName,
	})
	var rne *eksTypes.ResourceNotFoundException
	if errors.As(err, &rne) {
		log.Info().Msg("there is no EKS active for this cluster, and this is expected")
		return
	}
	if err != nil {
		t.Error(err)
	}
}

func TestAreEC2VolumesDestroyedIntegration(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	config := configs.ReadConfig()
	if len(config.AwsRegion) == 0 {
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
				strings.Contains(*volume.AvailabilityZone, config.AwsRegion) {

				isVolumeActive = true
			}
		}
	}

	if isVolumeActive {
		t.Error("it should not have active volumes for the current installation, but got at least one")
	}
}
