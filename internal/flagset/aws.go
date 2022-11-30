package flagset

import (
	"errors"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// DefineAWSFlags - define aws flags for CLI
func DefineAWSFlags(currentCommand *cobra.Command) {
	// AWS Flags
	currentCommand.Flags().String("s3-suffix", "", "unique identifier for s3 buckets")
	currentCommand.Flags().String("aws-assume-role", "", "instead of using AWS IAM user credentials, AWS AssumeRole feature generate role based credentials, more at https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html")
	currentCommand.Flags().Bool("aws-nodes-spot", false, "nodes spot on AWS EKS compute nodes")
	currentCommand.Flags().Bool("aws-nodes-graviton", false, "nodes Graviton on AWS EKS compute nodes, more info [https://aws.amazon.com/ec2/graviton/]")
	currentCommand.Flags().String("profile", "", "AWS profile located at ~/.aws/config")
	currentCommand.Flags().String("hosted-zone-name", "", "the domain to provision the kubefirst platform in")
	currentCommand.Flags().String("region", "", "the region to provision the cloud resources in")
}

type AwsFlags struct {
	Profile          string
	Region           string
	S3Suffix         string
	AssumeRole       string
	UseSpotInstance  bool
	UseNodesGraviton bool
	HostedZoneName   string
}

// ProcessAwsFlags - Read values of CLI parameters for aws flags
func ProcessAwsFlags(cmd *cobra.Command) (AwsFlags, error) {
	flags := AwsFlags{}
	// set profile
	profile, err := ReadConfigString(cmd, "profile")
	if err != nil {
		log.Println("unable to get profile values")
		return flags, err
	}
	viper.Set("aws.profile", profile)
	// propagate it to local environment
	err = os.Setenv("AWS_PROFILE", profile)
	if err != nil {
		log.Printf("unable to set environment variable AWS_PROFILE, error is: %v\n", err)
		return flags, err
	}
	log.Println("profile:", profile)
	flags.Profile = profile

	// set region
	region, err := ReadConfigString(cmd, "region")
	if err != nil {
		log.Println("unable to get region values from viper")
		return flags, err
	}
	viper.Set("aws.region", region)
	// propagate it to local environment
	err = os.Setenv("AWS_REGION", region)
	if err != nil {
		log.Printf("unable to set environment variable AWS_REGION, error is: %v\n", err)
		return flags, err
	}
	log.Println("region:", region)
	flags.Region = region

	nodesSpot, err := ReadConfigBool(cmd, "aws-nodes-spot")
	if err != nil {
		log.Println(err)
		return flags, err
	}
	viper.Set("aws.nodes_spot", nodesSpot)
	log.Println("aws.nodes_spot: ", nodesSpot)
	flags.UseSpotInstance = nodesSpot

	enableGraviton, err := ReadConfigBool(cmd, "aws-nodes-graviton")
	if err != nil {
		log.Println(err)
		return flags, err
	}
	viper.Set("aws.nodes_graviton", enableGraviton)
	log.Println("aws.nodes_graviton: ", enableGraviton)
	flags.UseNodesGraviton = enableGraviton

	bucketRand, err := ReadConfigString(cmd, "s3-suffix")
	if err != nil {
		log.Println(err)
		return flags, err
	}
	viper.Set("bucket.rand", bucketRand)
	flags.S3Suffix = bucketRand

	arnRole, err := ReadConfigString(cmd, "aws-assume-role")
	if err != nil {
		log.Println("unable to use the provided AWS IAM role for AssumeRole feature")
		return flags, err
	}
	viper.Set("aws.arn", arnRole)
	flags.AssumeRole = arnRole

	hostedZoneName, _ := ReadConfigString(cmd, "hosted-zone-name")
	if err != nil {
		return flags, err
	}
	viper.Set("aws.hostedzonename", hostedZoneName)
	flags.HostedZoneName = hostedZoneName

	err = validateAwsFlags()
	if err != nil {
		log.Println("Error validateAwsFlags:", err)
		return AwsFlags{}, err
	}

	return flags, nil
}

func validateAwsFlags() error {
	//Validation:
	//If you are changind this rules, please ensure to update:
	// internal/flagset/init_test.go
	if viper.GetString("cloud") != CloudAws {
		// To skip later validations
		// TODO: Create test scenarios for init
		log.Println("Skipping AWS Validation:", viper.GetString("cloud"))
		return nil
	}
	if len(viper.GetString("aws.hostedzonename")) < 1 {
		log.Println("Missing flag --hosted-zone-name for aws installation")
		return errors.New("missing flag --hosted-zone-name for an aws installation")
	}
	if len(viper.GetString("aws.region")) < 1 {
		log.Println("Missing flag --region for aws installation")
		return errors.New("missing flag --region for an aws installation")
	}
	if viper.GetString("aws.arn") == "" && viper.GetString("aws.profile") == "" {
		log.Println("aws.arn is empty", viper.GetString("aws.arn"))
		log.Println("aws.profile is empty", viper.GetString("aws.profile"))
		return errors.New("must provide profile or aws-assume-role argument for aws installations of kubefirst")
	}

	if viper.GetString("aws.arn") != "" && viper.GetString("aws.profile") != "" {
		log.Println("aws.arn is ", viper.GetString("aws.arn"))
		log.Println("aws.profile is: ", viper.GetString("aws.profile"))
		log.Println("must provide only one of these arguments: profile or aws-assume-role")
		return errors.New("must provide only one of these arguments: profile or aws-assume-role")
	}

	if viper.GetString("gitprovider") == "gitlab" && viper.GetBool("aws.nodes_graviton") {
		log.Println("GitLab only support x86 compute nodes")
		return errors.New("GitLab only support x86 compute nodes")
	}

	return nil
}
