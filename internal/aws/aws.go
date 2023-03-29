package aws

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/spf13/viper"
)

// TXTRecord stores Route53 TXT record data
type TXTRecord struct {
	Name          string
	Value         string
	SetIdentifier *string
	Weight        *int64
	TTL           int64
}

// ARecord stores Route53 A record data
type ARecord struct {
	Name        string
	RecordType  string
	TTL         *int64
	AliasTarget *route53Types.AliasTarget
}

var Conf AWSConfiguration = AWSConfiguration{
	Config: NewAwsV2(),
}

// Some systems fail to resolve TXT records, so try to use Google as a backup
var backupResolver = &net.Resolver{
	PreferGo: true,
	Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{
			Timeout: time.Millisecond * time.Duration(10000),
		}
		return d.DialContext(ctx, network, "8.8.8.8:53")
	},
}

func NewAwsV2() aws.Config {
	region := viper.GetString("flags.cloud-region")
	// todo these should also be supported flags
	profile := os.Getenv("AWS_PROFILE")

	awsClient, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		log.Panic().Msg("unable to create aws client")
	}

	return awsClient
}

// NewAws instantiate a new AWS configuration. This function is used to provide initial connection to AWS services.
// todo: update AWS functions in this file to work as methods of AWS struct
// example:
// DestroyBucketsInUse will have its function signature updated to (awsConfig AWSStruct) DestroyBucketsInUse(param type)
// and AWSStruct will be used as instanceOfAws.DestroyBucketsInUse(param type)
func NewAws() (aws.Config, error) {

	// todo these should also be supported flags
	region := os.Getenv("AWS_REGION")
	profile := os.Getenv("AWS_PROFILE")

	awsClient, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("unable to initialize AWS service, error: %v", err)
	}

	return awsClient, nil
}

// GetDNSInfo try to reach the provided hosted zone
func GetDNSInfo(hostedZoneName string) string {

	log.Info().Msg("GetDNSInfo (working...)")

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := NewAws()
	if err != nil {
		log.Warn().Msgf("failed to load configuration, error: %s", err)
	}
	// https://aws.github.io/aws-sdk-go-v2/docs/making-requests/#overriding-configuration
	route53Client := route53.NewFromConfig(awsConfig)
	hostedZones, err := route53Client.ListHostedZonesByName(
		context.Background(),
		&route53.ListHostedZonesByNameInput{
			DNSName: &hostedZoneName,
		},
	)
	if err != nil {
		log.Info().Msgf("oh no error on call %s", err)
	}

	var hostedZoneId string

	for _, zone := range hostedZones.HostedZones {

		if *zone.Name == fmt.Sprintf(`%s%s`, hostedZoneName, ".") {

			hostedZoneId = strings.Split(*zone.Id, "/")[2]

			log.Info().Msgf(`found entry for user submitted domain %s, using hosted zone id %s`, hostedZoneName, hostedZoneId)

			viper.Set("aws.hostedzonename", hostedZoneName)
			viper.Set("aws.hostedzoneid", hostedZoneId)
			if err = viper.WriteConfig(); err != nil {
				log.Warn().Msgf("%s", err)
			}
		}
	}
	log.Info().Msg("GetDNSInfo (done)")
	return hostedZoneId

}

// CreateBucket creates a bucket specified in the bucketName field, and use aws.region set on .kubefirst config file
func CreateBucket(dryRun bool, bucketName string) {

	log.Info().Msg("createBucketCalled")
	if dryRun {
		log.Info().Msgf("[#99] Dry-run mode, bucket creation skipped:  %s", bucketName)
		return
	}

	// todo: use method approach to avoid new AWS client initializations
	awsClient, err := NewAws()
	if err != nil {
		log.Warn().Msgf("failed to attempt bucket creation, error: %v ", err)
		os.Exit(1)
	}

	s3Client := s3.NewFromConfig(awsClient)

	log.Info().Msgf("creating bucket: %s", bucketName)

	regionName := viper.GetString("aws.region")
	log.Info().Msgf("region is %s", regionName)

	if regionName == "us-east-1" {
		_, err = s3Client.CreateBucket(
			context.Background(), &s3.CreateBucketInput{
				Bucket: &bucketName,
			})
	} else {
		_, err = s3Client.CreateBucket(
			context.Background(),
			&s3.CreateBucketInput{
				Bucket: &bucketName,
				CreateBucketConfiguration: &s3Types.CreateBucketConfiguration{
					LocationConstraint: s3Types.BucketLocationConstraint(regionName),
				},
			})
	}
	if err != nil {
		// todo: redo it using AWS SDK v2 using SDK types
		//if awsErr, ok := err.(awserr.Error); ok {
		//	switch awsErr.Code() {
		//	case s3.ErrCodeBucketAlreadyExists:
		//		log.Println("Bucket already exists " + bucketName)
		//		os.Exit(1)
		//	case s3.ErrCodeBucketAlreadyOwnedByYou:
		//		log.Println("Bucket already exists but OwnedByYou, the process will continue: " + bucketName)
		//	}
		//} else {
		//	log.Println("failed to create bucket "+bucketName, err.Error())
		//	os.Exit(1)
		//}
		log.Warn().Msgf("%s", err)
	}

	viper.Set(fmt.Sprintf("bucket.%s.created", bucketName), true)
	viper.Set(fmt.Sprintf("bucket.%s.name", bucketName), bucketName)
	if err = viper.WriteConfig(); err != nil {
		log.Warn().Msgf("%s", err)
	}
}
