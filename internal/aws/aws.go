package aws

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/cip8/autoname"
	"github.com/kubefirst/kubefirst/pkg"
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

// NewAws instantiate a new AWS configuration. This function is used to provide initial connection to AWS services.
// todo: update AWS functions in this file to work as methods of AWS struct
// example:
// DestroyBucketsInUse will have its function signature updated to (awsConfig AWSStruct) DestroyBucketsInUse(param type)
// and AWSStruct will be used as instanceOfAws.DestroyBucketsInUse(param type)
func NewAws() (aws.Config, error) {

	// tests doesnt have access to viper, for tests we get these values from the environment
	region := viper.GetString("aws.region")
	if len(region) == 0 {
		region = os.Getenv("AWS_REGION")
	}
	profile := viper.GetString("aws.profile")
	if len(profile) == 0 {
		profile = os.Getenv("AWS_PROFILE")
	}

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

func BucketRand(dryRun bool) {

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := NewAws()
	if err != nil {
		log.Warn().Msgf("%s", err)
	}

	s3Client := s3.NewFromConfig(awsConfig)

	randomName := viper.GetString("bucket.rand")
	if randomName == "" {
		randomName = strings.ReplaceAll(autoname.Generate(), "_", "-")
		viper.Set("bucket.rand", randomName)
	}

	buckets := strings.Fields("state-store argo-artifacts gitlab-backup chartmuseum")
	for _, bucket := range buckets {
		bucketExists := viper.GetBool(fmt.Sprintf("bucket.%s.created", bucket))
		if !bucketExists {
			bucketName := fmt.Sprintf("k1-%s-%s", bucket, randomName)

			log.Info().Msgf("creating %s bucket %s", bucket, bucketName)

			regionName := viper.GetString("aws.region")
			log.Info().Msgf("region is %s", regionName)
			if !dryRun {
				if regionName == "us-east-1" {
					_, err = s3Client.CreateBucket(
						context.Background(),
						&s3.CreateBucketInput{
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
					log.Warn().Msgf("failed to create bucket %s - %s", bucketName, err.Error())
					os.Exit(1)
				}

				versionConfigInput := &s3.PutBucketVersioningInput{
					Bucket: aws.String(bucketName),
					VersioningConfiguration: &s3Types.VersioningConfiguration{
						Status: s3Types.BucketVersioningStatusEnabled,
					},
				}

				_, err := s3Client.PutBucketVersioning(context.Background(), versionConfigInput)
				if err != nil {
					log.Panic().Msgf("error putting S3 versioning: %s", err)
				}
				PutTagKubefirstOnBuckets(bucketName, viper.GetString("cluster-name"))
			} else {
				log.Info().Msgf("[#99] Dry-run mode, bucket creation skipped:  %s", bucketName)
			}
			viper.Set(fmt.Sprintf("bucket.%s.created", bucket), true)
			viper.Set(fmt.Sprintf("bucket.%s.name", bucket), bucketName)
			if err = viper.WriteConfig(); err != nil {
				log.Warn().Msgf("%s", err)
			}
		}
		log.Info().Msgf("bucket %s exists", viper.GetString(fmt.Sprintf("bucket.%s.name", bucket)))
	}
}

// GetAccountInfo collect IAM and roles data. Collected data like (account id and ARN) are stored in viper.
func GetAccountInfo() {

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := NewAws()
	if err != nil {
		log.Panic().Msgf("failed to load configuration, error: %s", err)
	}

	stsClient := sts.NewFromConfig(awsConfig)
	iamCaller, err := stsClient.GetCallerIdentity(
		context.Background(),
		&sts.GetCallerIdentityInput{},
	)
	if err != nil {
		log.Panic().Msgf("error: could not get caller identity %s", err)
	}

	viper.Set("aws.accountid", *iamCaller.Account)
	viper.Set("aws.userarn", *iamCaller.Arn)
	if err = viper.WriteConfig(); err != nil {
		log.Warn().Msgf("%s", err)
	}
}

// TestHostedZoneLiveness check Route53 for liveness entry, and check if it's responding/live
func TestHostedZoneLiveness(dryRun bool, hostedZoneName, hostedZoneId string) bool {
	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, TestHostedZoneLiveness skipped.")
		return true
	}

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := NewAws()
	if err != nil {
		log.Warn().Msgf("failed to load configuration, error: %s", err)
	}

	// https://aws.github.io/aws-sdk-go-v2/docs/making-requests/#overriding-configuration
	route53Client := route53.NewFromConfig(awsConfig)

	// todo when checking to see if hosted zone exists print ns records for user to verity in dns registrar
	route53RecordName := fmt.Sprintf("kubefirst-liveness.%s", hostedZoneName)
	route53RecordValue := "domain record propagated"

	log.Info().Msgf("checking to see if record %s exists", route53RecordName)
	log.Info().Msgf("hostedZoneId %s", hostedZoneId)
	log.Info().Msgf("route53RecordName %s", route53RecordName)
	record, err := route53Client.ChangeResourceRecordSets(
		context.Background(),
		&route53.ChangeResourceRecordSetsInput{
			ChangeBatch: &route53Types.ChangeBatch{
				Changes: []route53Types.Change{
					{
						Action: "UPSERT",
						ResourceRecordSet: &route53Types.ResourceRecordSet{
							Name: aws.String(route53RecordName),
							Type: "TXT",
							ResourceRecords: []route53Types.ResourceRecord{
								{
									Value: aws.String(strconv.Quote(route53RecordValue)),
								},
							},
							TTL:           aws.Int64(10),
							Weight:        aws.Int64(100),
							SetIdentifier: aws.String("CREATE sanity check for kubefirst installation"),
						},
					},
				},
				Comment: aws.String("CREATE sanity check dns record."),
			},
			HostedZoneId: aws.String(hostedZoneId),
		})
	if err != nil {
		log.Warn().Msgf("%s", err)
		return false
	}
	log.Info().Msgf("record creation status is %s", record.ChangeInfo.Status)
	count := 0
	// todo need to exit after n number of minutes and tell them to check ns records
	// todo this logic sucks
	for count <= 100 {
		count++

		log.Info().Msgf("%s", route53RecordName)
		ips, err := net.LookupTXT(route53RecordName)
		if err != nil {
			ips, err = backupResolver.LookupTXT(context.Background(), route53RecordName)
		}

		log.Info().Msgf("%s", ips)

		if err != nil {
			log.Warn().Msgf("Could not get record name %s - waiting 10 seconds and trying again: \nerror: %s", route53RecordName, err)
			time.Sleep(10 * time.Second)
		} else {
			for _, ip := range ips {
				// todo check ip against route53RecordValue in some capacity so we can pivot the value for testing
				log.Info().Msgf("%s. in TXT record value: %s\n", route53RecordName, ip)
				count = 101
			}
		}
		if count == 100 {
			log.Panic().Msg("unable to resolve hosted zone dns record. please check your domain registrar")
		}
	}
	return true
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

// ListBucketsInUse list user active buckets, except the backup bucket
func ListBucketsInUse() []string {
	var bucketsInUse []string

	backupSSLBucket := "k1-" + viper.GetString("aws.hostedzonename")

	bucketsConfig := viper.AllKeys()
	for _, bucketKey := range bucketsConfig {
		if strings.HasPrefix(bucketKey, "bucket.") &&
			strings.HasSuffix(bucketKey, ".name") &&
			!strings.Contains(bucketKey, backupSSLBucket) {

			bucketName := viper.GetString(bucketKey)
			bucketsInUse = append(bucketsInUse, bucketName)
		}
	}
	return bucketsInUse
}

// DestroyBucketsInUse receives a list of user active buckets, and try to destroy them
func DestroyBucketsInUse(dryRun bool, executeConfirmation bool) error {
	if dryRun {
		log.Info().Msg("Skip: DestroyBucketsInUse - Dry-run mode")
		return nil
	}
	if !executeConfirmation {
		log.Info().Msg("Skip: DestroyBucketsInUse - Not provided confirmation")
		return nil
	}

	log.Info().Msg("Confirmed: DestroyBucketsInUse")

	for _, bucket := range ListBucketsInUse() {
		log.Info().Msgf("Deleting versions, objects and bucket: %s:", bucket)
		err := DestroyBucketObjectsAndVersions(bucket, viper.GetString("aws.region"))
		if err != nil {
			return errors.New("error deleting bucket/objects/version, the resources may have already been removed, please re-run without flag --destroy-buckets and check on console")
		}
	}
	return nil
}

// AssumeRole receives a AWS IAM Role, and instead of using regular AWS credentials, it generates new AWS credentials
// based on the provided role. New AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, and AWS_SESSION_TOKEN are provided. The
// new AWS credentials has expiration time set.
func AssumeRole(roleArn string) error {

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := NewAws()
	if err != nil {
		return err
	}

	stsClient := sts.NewFromConfig(awsConfig)

	provider := stscreds.NewAssumeRoleProvider(stsClient, roleArn)

	awsConfig.Credentials = aws.NewCredentialsCache(provider)

	credentials, err := awsConfig.Credentials.Retrieve(context.Background())
	if err != nil {
		return err
	}

	// update AWS keys
	if err := os.Setenv("AWS_ACCESS_KEY_ID", credentials.AccessKeyID); err != nil {
		fmt.Printf("unable to set AWS_ACCESS_KEY_ID environment variable. Err: %v", err)
	}

	if err := os.Setenv("AWS_SECRET_ACCESS_KEY", credentials.SecretAccessKey); err != nil {
		fmt.Printf("unable to set AWS_SECRET_ACCESS_KEY environment variable. Err: %v", err)
	}

	if err := os.Setenv("AWS_SESSION_TOKEN", credentials.SessionToken); err != nil {
		fmt.Printf("unable to set AWS_SESSION_TOKEN environment variable. Err: %v", err)
	}

	return nil
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

// UploadFile receives a bucket name, a file name and upload it to AWS S3.
func UploadFile(bucketName string, remoteFilename string, localFilename string) error {

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := NewAws()
	if err != nil {
		log.Warn().Msgf("%s", err)
	}

	s3Client := manager.NewUploader(s3.NewFromConfig(awsConfig))

	f, err := os.Open(localFilename)
	if err != nil {
		return fmt.Errorf("failed to open file %q, %v", localFilename, err)
	}

	// Upload file to S3
	result, err := s3Client.Upload(
		context.Background(),
		&s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(remoteFilename),
			Body:   f,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to upload file, %v", err)
	}
	log.Info().Msgf("file succesfully uploaded to, %s", result.Location)
	return nil
}

func DownloadBucket(bucket string, destFolder string) error {

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := NewAws()
	if err != nil {
		log.Warn().Msgf("%s", err)
	}

	s3Client := s3.NewFromConfig(awsConfig)

	downloader := manager.NewDownloader(s3.NewFromConfig(awsConfig))

	log.Info().Msg("Listing the objects in the bucket:")
	listObjsResponse, err := s3Client.ListObjectsV2(context.Background(),
		&s3.ListObjectsV2Input{
			Bucket: aws.String(bucket),
			Prefix: aws.String(""),
		})

	if err != nil {
		log.Warn().Msgf("Error DownloadBucket: %s", err)
		return errors.New("couldn't list bucket contents")
	}

	for _, object := range listObjsResponse.Contents {
		log.Info().Msgf("%s (%d bytes, class %v) ", *object.Key, object.Size, object.StorageClass)

		f, err := pkg.CreateFullPath(filepath.Join(destFolder, *object.Key))
		if err != nil {
			log.Warn().Msgf("Error DownloadBucket: %s", err)
			return fmt.Errorf("failed to create file %q, %v", *object.Key, err)
		}

		// Write the contents of S3 Object to the file
		_, err = downloader.Download(context.Background(),
			f,
			&s3.GetObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(*object.Key),
			})
		if err != nil {
			log.Warn().Msgf("Error DownloadBucket: %s", err)
			return fmt.Errorf("failed to download file, %v", err)
		}
		// close file immediately
		if err = f.Close(); err != nil {
			log.Warn().Msgf("Error DownloadBucket: %s", err)
			return err
		}
	}
	return nil
}

func PutTagKubefirstOnBuckets(bucketName string, clusterName string) {

	log.Info().Msgf("tagging bucket... %s:%s", bucketName, clusterName)

	awsConfig, err := NewAws()
	if err != nil {
		log.Warn().Msgf("%s", err)
	}
	s3Client := s3.NewFromConfig(awsConfig)

	input := &s3.PutBucketTaggingInput{
		Bucket: aws.String(bucketName),
		Tagging: &s3Types.Tagging{
			TagSet: []s3Types.Tag{
				{
					Key:   aws.String("Provisioned-by"),
					Value: aws.String("Kubefirst"),
				},
				{
					Key:   aws.String("ClusterName"),
					Value: aws.String(clusterName),
				},
			},
		},
	}

	_, err = s3Client.PutBucketTagging(context.Background(), input)
	if err != nil {
		// todo: redo it using AWS SDK v2 using SDK types
		//if aerr, ok := err.(awserr.Error); ok {
		//	log.Println(aerr.Error())
		//} else {
		//	log.Println(err.Error())
		//}
		//return
		log.Warn().Msgf("%s", err)
		return
	}
	log.Info().Msgf("Bucket: %s tagged successfully", bucketName)
}

func DestroyBucketObjectsAndVersions(bucket, region string) error {

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := NewAws()
	if err != nil {
		log.Warn().Msgf("Failed to load config: %v", err)
		return err
	}

	client := s3.NewFromConfig(awsConfig)

	deleteObject := func(bucket, key, versionId *string) {
		log.Info().Msgf("Object: %s/%s", *key, aws.ToString(versionId))
		_, err := client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
			Bucket:    bucket,
			Key:       key,
			VersionId: versionId,
		})
		if err != nil {
			log.Warn().Msgf("Failed to delete object: %v", err)
		}
	}

	in := &s3.ListObjectsV2Input{Bucket: &bucket}
	for {
		out, err := client.ListObjectsV2(context.Background(), in)
		if err != nil {
			bucketNotFound := strings.Contains(err.Error(), "StatusCode: 404")
			if bucketNotFound {
				log.Warn().Msgf("%s has already been removed, proceeding with clean...", bucket)
			} else {
				log.Warn().Msgf("Failed to list objects: %v", err)
				return err
			}
		}

		if out != nil {
			for _, item := range out.Contents {
				deleteObject(&bucket, item.Key, nil)
			}
		}

		if out != nil && out.IsTruncated {
			in.ContinuationToken = out.ContinuationToken
		} else {
			break
		}
	}

	inVer := &s3.ListObjectVersionsInput{Bucket: &bucket}
	for {
		out, err := client.ListObjectVersions(context.Background(), inVer)
		if err != nil {
			bucketNotFound := strings.Contains(err.Error(), "StatusCode: 404")
			if bucketNotFound {
				log.Warn().Msgf("%s has already been removed, proceeding with clean...", bucket)
			} else {
				log.Warn().Msgf("Failed to list version objects: %v", err)
				return err
			}
		}

		if out != nil {
			for _, item := range out.DeleteMarkers {
				deleteObject(&bucket, item.Key, item.VersionId)
			}

			for _, item := range out.Versions {
				deleteObject(&bucket, item.Key, item.VersionId)
			}
		}

		if out != nil && out.IsTruncated {
			inVer.VersionIdMarker = out.NextVersionIdMarker
			inVer.KeyMarker = out.NextKeyMarker
		} else {
			break
		}
	}

	_, err = client.DeleteBucket(context.Background(), &s3.DeleteBucketInput{Bucket: &bucket})
	if err != nil {
		log.Warn().Msgf("Failed to delete bucket: %v", err)
	}
	return nil
}

// DownloadS3File receives a bucket name, filename and download the file at AWS S3
func DownloadS3File(bucketName string, filename string) error {

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := NewAws()
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	s3Client := manager.NewDownloader(s3.NewFromConfig(awsConfig))
	numBytes, err := s3Client.Download(
		context.Background(),
		file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(filename),
		},
	)
	if err != nil {
		return err
	}

	log.Info().Msgf("Downloaded file: %s, file size(bytes): %v", file.Name(), numBytes)

	return nil
}

// Route53GetHostedZoneId translates a Hosted Zone into Name into a Hosted Zone Id.
func Route53GetHostedZoneId(hostedZone string) (string, error) {

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := NewAws()
	if err != nil {
		return "", err
	}

	route53Client := route53.NewFromConfig(awsConfig)

	routes, err := route53Client.ListHostedZonesByName(
		context.Background(),
		&route53.ListHostedZonesByNameInput{
			DNSName: &hostedZone,
		})
	if err != nil {
		return "", err
	}

	var hostedZoneId string
	for _, dnsEntry := range routes.HostedZones {
		if *dnsEntry.Name == hostedZone+"." {
			hostedZoneId = *dnsEntry.Id
		}
	}

	return hostedZoneId, nil
}

// Route53ListTXTRecords retrieve all DNS TXT record type for a specific Host Zone Id.
func Route53ListTXTRecords(hostedZoneId string) ([]TXTRecord, error) {

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := NewAws()
	if err != nil {
		return []TXTRecord{}, err
	}

	route53Client := route53.NewFromConfig(awsConfig)

	log.Info().Msgf("hosted zone found! Hosted Zone id: %s", hostedZoneId)

	recordSets, err := route53Client.ListResourceRecordSets(context.Background(), &route53.ListResourceRecordSetsInput{
		HostedZoneId: &hostedZoneId,
	})
	if err != nil {
		return []TXTRecord{}, err
	}

	var txtRecords []TXTRecord

	for _, recordSet := range recordSets.ResourceRecordSets {
		log.Info().Msgf("Record Name: %s", *recordSet.Name)

		if recordSet.Type == route53Types.RRTypeTxt {
			for _, resourceRecord := range recordSet.ResourceRecords {

				if recordSet.SetIdentifier != nil && recordSet.Weight != nil {
					record := TXTRecord{
						Name:          *recordSet.Name,
						Value:         *resourceRecord.Value,
						SetIdentifier: recordSet.SetIdentifier,
						TTL:           *recordSet.TTL,
						Weight:        recordSet.Weight,
					}
					txtRecords = append(txtRecords, record)
					continue
				}

				record := TXTRecord{
					Name:  *recordSet.Name,
					Value: *resourceRecord.Value,
					TTL:   *recordSet.TTL,
				}
				txtRecords = append(txtRecords, record)
			}
		}
	}

	return txtRecords, nil
}

// Route53ListARecords retrieve all DNS A record type for a specific Host Zone Id.
func Route53ListARecords(hostedZoneId string) ([]ARecord, error) {

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := NewAws()
	if err != nil {
		return []ARecord{}, err
	}

	route53Client := route53.NewFromConfig(awsConfig)

	recordSets, err := route53Client.ListResourceRecordSets(context.Background(), &route53.ListResourceRecordSetsInput{
		HostedZoneId: &hostedZoneId,
	})
	if err != nil {
		return []ARecord{}, err
	}

	var aRecords []ARecord

	for _, recordSet := range recordSets.ResourceRecordSets {

		if recordSet.Type == route53Types.RRTypeA {

			record := ARecord{
				Name:       *recordSet.Name,
				RecordType: "A",
				AliasTarget: &route53Types.AliasTarget{
					HostedZoneId:         recordSet.AliasTarget.HostedZoneId,
					DNSName:              recordSet.AliasTarget.DNSName,
					EvaluateTargetHealth: true,
				},
			}
			aRecords = append(aRecords, record)
		}
	}

	return aRecords, nil

}

// Route53DeleteTXTRecords receives a list of DNS TXT records []TXTRecord, and delete the records contained in the list.
// todo: improve logging
// todo: record deletion should use a function that receives the TXT parameters, and deletes the record. Not doing it
// now since handler/methods needs to be implemented first.
func Route53DeleteTXTRecords(
	hostedZoneId string,
	hostedZoneName string,
	keepLivenessRecord bool,
	txtRecords []TXTRecord,
) error {

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := NewAws()
	if err != nil {
		return err
	}

	route53Client := route53.NewFromConfig(awsConfig)

	livenessRecordName := fmt.Sprintf("%s.%s.", "kubefirst-liveness", hostedZoneName)

	for _, record := range txtRecords {

		if keepLivenessRecord && record.Name == livenessRecordName {
			log.Info().Msgf("%s record not deleted\n", record.Name)
			continue
		}

		log.Info().Msgf("deleting TXT record... %s", record.Name)

		//this deletes a TXT record
		if record.SetIdentifier != nil && record.Weight != nil {
			_, err = route53Client.ChangeResourceRecordSets(context.Background(), &route53.ChangeResourceRecordSetsInput{
				ChangeBatch: &route53Types.ChangeBatch{
					Changes: []route53Types.Change{
						{
							Action: "DELETE",
							ResourceRecordSet: &route53Types.ResourceRecordSet{
								Name:          &record.Name,
								Type:          "TXT",
								TTL:           &record.TTL,
								SetIdentifier: record.SetIdentifier,
								Weight:        record.Weight,
								ResourceRecords: []route53Types.ResourceRecord{
									{
										Value: &record.Value,
									},
								},
							},
						},
					},
				},
				HostedZoneId: &hostedZoneId,
			})
			if err != nil {
				return err
			}
		} else {
			_, err = route53Client.ChangeResourceRecordSets(context.Background(), &route53.ChangeResourceRecordSetsInput{
				ChangeBatch: &route53Types.ChangeBatch{
					Changes: []route53Types.Change{
						{
							Action: "DELETE",
							ResourceRecordSet: &route53Types.ResourceRecordSet{
								Name: &record.Name,
								Type: "TXT",
								TTL:  &record.TTL,
								ResourceRecords: []route53Types.ResourceRecord{
									{
										Value: &record.Value,
									},
								},
							},
						},
					},
				},
				HostedZoneId: &hostedZoneId,
			})
			if err != nil {
				return err
			}
		}
		log.Info().Msgf("Route53 TXT record deleted: %q", record.Name)
	}

	return nil

}

// Route53DeleteARecords receives a slice of DNS A Record []ARecord, and delete all Route 53 records contained in the
// list.
func Route53DeleteARecords(hostedZoneId string, aRecords []ARecord) error {

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := NewAws()
	if err != nil {
		return err
	}

	route53Client := route53.NewFromConfig(awsConfig)

	for _, record := range aRecords {

		_, err := route53Client.ChangeResourceRecordSets(context.Background(), &route53.ChangeResourceRecordSetsInput{
			ChangeBatch: &route53Types.ChangeBatch{
				Changes: []route53Types.Change{
					{
						Action: "DELETE",
						ResourceRecordSet: &route53Types.ResourceRecordSet{
							Name: &record.Name,
							Type: "A",
							AliasTarget: &route53Types.AliasTarget{
								HostedZoneId:         record.AliasTarget.HostedZoneId,
								DNSName:              record.AliasTarget.DNSName,
								EvaluateTargetHealth: true,
							},
						},
					},
				},
			},
			HostedZoneId: &hostedZoneId,
		})
		if err != nil {
			return err
		}

		log.Info().Msgf("Route53 A record deleted: %q", record.Name)
	}

	return nil
}

// Route53DeleteHostedZone deletes the HostedZone. Route53DeleteHostedZone can be called only when there are no hosted
// zones records available (except NS and SOA records). When there are still hosted zone records available, the deletion
// will fail with the error description coming from AWS library.
func Route53DeleteHostedZone(hostedZoneId string, hostedZoneName string) error {
	awsConfig, err := NewAws()

	if err != nil {
		return err
	}

	route53Client := route53.NewFromConfig(awsConfig)

	hostedZoneInput := route53.DeleteHostedZoneInput{
		Id: &hostedZoneId,
	}

	log.Info().Msgf("trying to delete hosted zone id %q, hosted zone name %q", hostedZoneId, hostedZoneName)

	_, err = route53Client.DeleteHostedZone(context.Background(), &hostedZoneInput)
	if err != nil {
		return err
	}

	log.Info().Msgf("deleted hosted zone id %q, hosted zone name %q", hostedZoneId, hostedZoneName)

	return nil
}

func ProfileInjection(envs *map[string]string) {
	currentRole := viper.GetString("aws.arn")
	if currentRole == "" {
		(*envs)["AWS_PROFILE"] = viper.GetString("aws.profile")
	} else {
		log.Info().Msg("Skipping AWS Profile loading due Assume Role...")
	}
}

func DestroyLoadBalancerByName(elbName string) error {
	awsConfig, err := NewAws()
	if err != nil {
		log.Warn().Msgf("Failed to load config: %v", err)
		return err
	}
	loadBalancerNameString := string(elbName)

	if len(loadBalancerNameString) > 0 {
		loadBalancerClient := elasticloadbalancing.NewFromConfig(awsConfig)

		loadBalancerInput := elasticloadbalancing.DeleteLoadBalancerInput{
			LoadBalancerName: &loadBalancerNameString,
		}

		log.Info().Msgf("trying to delete load balancer %s", loadBalancerNameString)

		_, err = loadBalancerClient.DeleteLoadBalancer(context.Background(), &loadBalancerInput)

		if err != nil {
			return err
		}

		log.Info().Msgf("deleted load balancer %s", loadBalancerNameString)
	}

	return nil
}

func DestroySecurityGroupById(securityGroupId string) error {
	// todo: use method approach to avoid new AWS client initializations
	awsRegion := viper.GetString("aws.region")
	awsProfile := viper.GetString("aws.profile")
	awsConfig, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(awsRegion),
		config.WithSharedConfigProfile(awsProfile),
	)
	if err != nil {
		log.Warn().Msgf("error: %s", err)
	}

	if len(securityGroupId) > 0 {
		securityGroupClient := ec2.NewFromConfig(awsConfig)

		securityGroupInput := ec2.DeleteSecurityGroupInput{

			GroupId: aws.String(securityGroupId),
		}

		log.Info().Msgf("trying to delete security group %s", securityGroupId)

		_, err = securityGroupClient.DeleteSecurityGroup(context.Background(), &securityGroupInput)
		if err != nil {
			return err
		}

		log.Info().Msgf("deleted security group %s", securityGroupId)
	}

	return nil
}

func DestroySecurityGroupByName(securityGroupName string) error {
	// todo: use method approach to avoid new AWS client initializations
	awsRegion := viper.GetString("aws.region")
	awsProfile := viper.GetString("aws.profile")
	awsConfig, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(awsRegion),
		config.WithSharedConfigProfile(awsProfile),
	)
	if err != nil {
		log.Warn().Msgf("error: %s ", err)
	}

	if len(securityGroupName) > 0 {
		securityGroupClient := ec2.NewFromConfig(awsConfig)

		securityGroupInput := ec2.DeleteSecurityGroupInput{

			GroupName: &securityGroupName,
		}

		log.Info().Msgf("trying to delete security group %s", securityGroupName)

		_, err = securityGroupClient.DeleteSecurityGroup(context.Background(), &securityGroupInput)
		if err != nil {
			return err
		}

		log.Info().Msgf("deleted security group %s", securityGroupName)
	}

	return nil
}
func DestroyLoadBalancer(clusterName string) error {
	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := NewAws()
	if err != nil {
		log.Warn().Msgf("Failed to load config: %v", err)
		return err
	}

	searchLoadBalancerCmd := fmt.Sprintf("for i in $(aws elb describe-load-balancers | jq -r '.LoadBalancerDescriptions[].LoadBalancerName');"+
		"do aws elb describe-tags --load-balancer-names \"$i\" | jq -ce '.TagDescriptions[].Tags[] | select( .Key == \"kubernetes.io/cluster/%s\" and .Value == \"owned\")' &&"+
		"echo \"$i\" ;done | sed -n '2 p' | tr -d '\n\r'", clusterName)

	loadBalancerName, err := exec.Command("bash", "-c", searchLoadBalancerCmd).Output()
	if err != nil {
		log.Panic().Msgf("error: could not read load balancer name: %s", err)
	}

	loadBalancerNameString := string(loadBalancerName)

	if len(loadBalancerNameString) > 0 {
		loadBalancerClient := elasticloadbalancing.NewFromConfig(awsConfig)

		loadBalancerInput := elasticloadbalancing.DeleteLoadBalancerInput{
			LoadBalancerName: &loadBalancerNameString,
		}

		log.Info().Msgf("trying to delete load balancer %s", loadBalancerNameString)

		_, err = loadBalancerClient.DeleteLoadBalancer(context.Background(), &loadBalancerInput)

		if err != nil {
			return err
		}

		log.Info().Msgf("deleted load balancer %s", loadBalancerNameString)
	}

	return nil
}

func DestroySecurityGroup(clusterName string) error {
	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := NewAws()
	if err != nil {
		log.Warn().Msgf("Failed to load config: %v", err)
		return err
	}

	searchSecurityGroupCmd := fmt.Sprintf("aws ec2 describe-security-groups --filters Name=tag:kubernetes.io/cluster/%s,Values=owned "+
		"| jq -r '.SecurityGroups[].GroupId' | tr -d '\n\r'", clusterName)

	securityGroupId, err := exec.Command("bash", "-c", searchSecurityGroupCmd).Output()
	if err != nil {
		log.Panic().Msgf("error: could not read security group id: %s", err)
	}

	securityGroupIdString := string(securityGroupId)

	if len(securityGroupIdString) > 0 {
		securityGroupClient := ec2.NewFromConfig(awsConfig)

		securityGroupInput := ec2.DeleteSecurityGroupInput{
			GroupId: &securityGroupIdString,
		}

		log.Info().Msgf("trying to delete security group %s", securityGroupIdString)

		_, err = securityGroupClient.DeleteSecurityGroup(context.Background(), &securityGroupInput)
		if err != nil {
			return err
		}

		log.Info().Msgf("deleted security group %s", securityGroupIdString)
	}

	return nil
}

func GetELBDetails(ingressHost string) (string, string, error) {
	elb := strings.Split(ingressHost, "-")[0]
	securityGroup := "k8s-elb-" + elb

	return elb, securityGroup, nil

}

func GetVPCIdByClusterName(clusterName string) string {
	awsConfig, err := NewAws()
	if err != nil {
		log.Warn().Msgf("Failed to load config: %v", err)
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
		log.Warn().Msgf("%v", err)
	}

	if len(vpcData.Vpcs) > 0 {
		log.Info().Msgf("there is no VPC for the cluster %q", clusterName)
	}

	for _, v := range vpcData.Vpcs {
		vpcId := aws.ToString(v.VpcId)
		if v.State == "available" {
			//it is only expected to have 1 vpc per cluster name
			log.Info().Msgf("there is a VPC for the %q cluster, the vpcID is %s", clusterName, vpcId)
			return vpcId
		}
	}
	return ""
}

// GetELBByClusterName return the elb name and its security groups
func GetELBByClusterName(clusterName string) (string, []string) {
	awsConfig, err := NewAws()
	if err != nil {
		log.Warn().Msgf("Failed to load config: %v", err)
	}

	loadBalancerClient := elasticloadbalancing.NewFromConfig(awsConfig)

	elbs, err := loadBalancerClient.DescribeLoadBalancers(context.Background(), &elasticloadbalancing.DescribeLoadBalancersInput{})
	if err != nil {
		log.Warn().Msgf("%v", err)
	}
	if len(elbs.LoadBalancerDescriptions) > 0 {
		log.Info().Msgf("there is no ELB for the cluster %s", clusterName)
	}

	for _, elb := range elbs.LoadBalancerDescriptions {
		elbName := aws.ToString(elb.LoadBalancerName)
		tags, err := loadBalancerClient.DescribeTags(context.Background(), &elasticloadbalancing.DescribeTagsInput{
			LoadBalancerNames: []string{elbName},
		})
		if err != nil {
			log.Warn().Msgf("%v", err)
		}
		for _, tagDesc := range tags.TagDescriptions {
			for _, tag := range tagDesc.Tags {
				key := aws.ToString(tag.Key)
				value := aws.ToString(tag.Value)
				if value == "owned" && key == fmt.Sprintf("kubernetes.io/cluster/%s", clusterName) {
					log.Info().Msgf("Match tag: %s - %s ", key, value)
					log.Info().Msgf("Match  ELB Name: %s", elbName)
					log.Info().Msgf("Match ELB SG: %s", elb.SecurityGroups)
					//found the right ELB
					return elbName, elb.SecurityGroups
				}

			}
		}
	}
	return "", []string{""}
}
