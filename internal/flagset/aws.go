package flagset

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func DefineAWSFlags(currentCommand *cobra.Command) {
	// AWS Flags
	currentCommand.Flags().String("s3-suffix", "", "unique identifier for s3 buckets")
	currentCommand.Flags().String("aws-assume-role", "", "instead of using AWS IAM user credentials, AWS AssumeRole feature generate role based credentials, more at https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html")
	currentCommand.Flags().Bool("aws-nodes-spot", false, "nodes spot on AWS EKS compute nodes")
	currentCommand.Flags().String("profile", "default", "AWS profile located at ~/.aws/config")
	currentCommand.Flags().String("hosted-zone-name", "", "the domain to provision the kubefirst platform in")
	currentCommand.Flags().String("region", "eu-west-1", "the region to provision the cloud resources in")
}

type AwsFlags struct {
	Profile         string
	Region          string
	S3Suffix        string
	AssumeRole      string
	UseSpotInstance bool
	HostedZoneName  string
}

func ProcessAwsFlags(cmd *cobra.Command) (AwsFlags, error) {
	flags := AwsFlags{}
	// set profile
	profile, err := cmd.Flags().GetString("profile")
	if err != nil {
		log.Println("unable to get profile values")
		return flags, err
	}
	viper.Set("aws.profile", profile)
	// propagate it to local environment
	err = os.Setenv("AWS_PROFILE", profile)
	if err != nil {
		log.Println("unable to set environment variable AWS_PROFILE, error is: %v", err)
		return flags, err
	}
	log.Println("profile:", profile)
	flags.Profile = profile

	// set region
	region, err := cmd.Flags().GetString("region")
	if err != nil {
		log.Println("unable to get region values from viper")
		return flags, err
	}
	viper.Set("aws.region", region)
	// propagate it to local environment
	err = os.Setenv("AWS_REGION", region)
	if err != nil {
		log.Println("unable to set environment variable AWS_REGION, error is: %v", err)
		return flags, err
	}
	log.Println("region:", region)
	flags.Region = region

	nodesSpot, err := cmd.Flags().GetBool("aws-nodes-spot")
	if err != nil {
		log.Println(err)
		return flags, err
	}
	viper.Set("aws.nodes_spot", nodesSpot)
	log.Println("aws.nodes_spot: ", nodesSpot)
	flags.UseSpotInstance = nodesSpot

	bucketRand, err := cmd.Flags().GetString("s3-suffix")
	if err != nil {
		log.Println(err)
		return flags, err
	}
	viper.Set("bucket.rand", bucketRand)
	flags.S3Suffix = bucketRand

	arnRole, err := cmd.Flags().GetString("aws-assume-role")
	if err != nil {
		log.Println("unable to use the provided AWS IAM role for AssumeRole feature")
		return flags, err
	}
	viper.Set("aws.arn", arnRole)
	flags.AssumeRole = arnRole

	hostedZoneName, _ := cmd.Flags().GetString("hosted-zone-name")
	if err != nil {
		return flags, err
	}
	viper.Set("aws.hostedzonename", hostedZoneName)
	flags.HostedZoneName = hostedZoneName

	return flags, nil
}
