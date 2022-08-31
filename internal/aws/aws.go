package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/cip8/autoname"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

// newAws instantiate a new AWS configuration. This function is used to provide initial connection to AWS services.
// todo: update AWS functions in this file to work as methods of AWS struct, example:
// DestroyBucketsInUse will have its function signature updated to (awsConfig AWSStruct) DestroyBucketsInUse(param type)
// and AWSStruct will be used as instanceOfAws.DestroyBucketsInUse(param type)
func newAws() (aws.Config, error) {

	region := viper.GetString("aws.region")
	profile := viper.GetString("aws.profile")
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
	awsConfig, err := newAws()
	if err != nil {
		log.Println(err)
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

			log.Println("creating", bucket, "bucket", bucketName)

			regionName := viper.GetString("aws.region")
			log.Println("region is ", regionName)
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
					log.Println("failed to create bucket "+bucketName, err.Error())
					os.Exit(1)
				}

				versionConfigInput := &s3.PutBucketVersioningInput{
					Bucket: aws.String(bucketName),
					VersioningConfiguration: &s3Types.VersioningConfiguration{
						Status: s3Types.BucketVersioningStatusEnabled,
					},
				}
				log.Printf("[DEBUG] S3 put bucket versioning: %#v", versionConfigInput)
				_, err := s3Client.PutBucketVersioning(context.Background(), versionConfigInput)
				if err != nil {
					log.Panicf("Error putting S3 versioning: %s", err)
				}
				PutTagKubefirstOnBuckets(bucketName, viper.GetString("cluster-name"))
			} else {
				log.Printf("[#99] Dry-run mode, bucket creation skipped:  %s", bucketName)
			}
			viper.Set(fmt.Sprintf("bucket.%s.created", bucket), true)
			viper.Set(fmt.Sprintf("bucket.%s.name", bucket), bucketName)
			if err = viper.WriteConfig(); err != nil {
				log.Println(err)
			}
		}
		log.Printf("bucket %s exists", viper.GetString(fmt.Sprintf("bucket.%s.name", bucket)))
	}
}

// GetAccountInfo collect IAM and roles data. Collected data like (account id and ARN) are stored in viper.
func GetAccountInfo() {

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := newAws()
	if err != nil {
		log.Panicf("failed to load configuration, error: %s", err)
	}

	stsClient := sts.NewFromConfig(awsConfig)
	iamCaller, err := stsClient.GetCallerIdentity(
		context.Background(),
		&sts.GetCallerIdentityInput{},
	)
	if err != nil {
		log.Panicf("error: could not get caller identity %s", err)
	}

	viper.Set("aws.accountid", *iamCaller.Account)
	viper.Set("aws.userarn", *iamCaller.Arn)
	if err = viper.WriteConfig(); err != nil {
		log.Println(err)
	}
}

// TestHostedZoneLiveness check Route53 for liveness entry, and check if it's responding/live
func TestHostedZoneLiveness(dryRun bool, hostedZoneName, hostedZoneId string) bool {
	if dryRun {
		log.Printf("[#99] Dry-run mode, TestHostedZoneLiveness skipped.")
		return true
	}

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := newAws()
	if err != nil {
		log.Println("failed to load configuration, error:", err)
	}

	// https://aws.github.io/aws-sdk-go-v2/docs/making-requests/#overriding-configuration
	route53Client := route53.NewFromConfig(awsConfig)

	// todo when checking to see if hosted zone exists print ns records for user to verity in dns registrar
	route53RecordName := fmt.Sprintf("kubefirst-liveness.%s", hostedZoneName)
	route53RecordValue := "domain record propagated"

	log.Println("checking to see if record", route53RecordName, "exists")
	log.Println("hostedZoneId", hostedZoneId)
	log.Println("route53RecordName", route53RecordName)
	record, err := route53Client.ChangeResourceRecordSets(
		context.Background(),
		&route53.ChangeResourceRecordSetsInput{
			ChangeBatch: &types.ChangeBatch{
				Changes: []types.Change{
					{
						Action: "UPSERT",
						ResourceRecordSet: &types.ResourceRecordSet{
							Name: aws.String(route53RecordName),
							Type: "TXT",
							ResourceRecords: []types.ResourceRecord{
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
		log.Println(err)
		return false
	}
	log.Println("record creation status is ", record.ChangeInfo.Status)
	count := 0
	// todo need to exit after n number of minutes and tell them to check ns records
	// todo this logic sucks
	for count <= 100 {
		count++

		log.Println(route53RecordName)
		ips, err := net.LookupTXT(route53RecordName)

		log.Println(ips)

		if err != nil {
			log.Println(fmt.Sprintf("Could not get record name %s - waiting 10 seconds and trying again: \nerror: %s", route53RecordName, err))
			time.Sleep(10 * time.Second)
		} else {
			for _, ip := range ips {
				// todo check ip against route53RecordValue in some capacity so we can pivot the value for testing
				log.Println(fmt.Sprintf("%s. in TXT record value: %s\n", route53RecordName, ip))
				count = 101
			}
		}
		if count == 100 {
			log.Panicf("unable to resolve hosted zone dns record. please check your domain registrar")
		}
	}
	return true
}

// GetDNSInfo try to reach the provided hosted zone
func GetDNSInfo(hostedZoneName string) string {

	log.Println("GetDNSInfo (working...)")

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := newAws()
	if err != nil {
		log.Println("failed to load configuration, error:", err)
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
		log.Println("oh no error on call", err)
	}

	var hostedZoneId string

	for _, zone := range hostedZones.HostedZones {

		if *zone.Name == fmt.Sprintf(`%s%s`, hostedZoneName, ".") {

			hostedZoneId = strings.Split(*zone.Id, "/")[2]

			log.Printf(`found entry for user submitted domain %s, using hosted zone id %s`, hostedZoneName, hostedZoneId)

			viper.Set("aws.hostedzonename", hostedZoneName)
			viper.Set("aws.hostedzoneid", hostedZoneId)
			if err = viper.WriteConfig(); err != nil {
				log.Println(err)
			}
		}
	}
	log.Println("GetDNSInfo (done)")
	return hostedZoneId

}

// ListBucketsInUse list user active buckets
func ListBucketsInUse() []string {
	var bucketsInUse []string
	bucketsConfig := viper.AllKeys()
	for _, bucketKey := range bucketsConfig {
		if strings.HasPrefix(bucketKey, "bucket.") &&
			strings.HasSuffix(bucketKey, ".name") &&
			!strings.Contains(bucketKey, "state-store") {
			bucketName := viper.GetString(bucketKey)
			bucketsInUse = append(bucketsInUse, bucketName)
		}
	}
	return bucketsInUse
}

// DestroyBucketsInUse receives a list of user active buckets, and try to destroy them
func DestroyBucketsInUse(dryRun bool, executeConfirmation bool) error {
	if dryRun {
		log.Println("Skip: DestroyBucketsInUse - Dry-run mode")
		return nil
	}
	if !executeConfirmation {
		log.Println("Skip: DestroyBucketsInUse - Not provided confirmation")
		return nil
	}

	log.Println("Confirmed: DestroyBucketsInUse")

	for _, bucket := range ListBucketsInUse() {
		log.Printf("Deleting versions, objects and bucket: %s:", bucket)
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
	awsConfig, err := newAws()
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

	log.Println("createBucketCalled")
	if dryRun {
		log.Printf("[#99] Dry-run mode, bucket creation skipped:  %s", bucketName)
		return
	}

	// todo: use method approach to avoid new AWS client initializations
	awsClient, err := newAws()
	if err != nil {
		log.Printf("failed to attempt bucket creation, error: %v ", err)
		os.Exit(1)
	}

	s3Client := s3.NewFromConfig(awsClient)

	log.Println("creating bucket: ", bucketName)

	regionName := viper.GetString("aws.region")
	log.Println("region is ", regionName)

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
		log.Println(err)
	}

	viper.Set(fmt.Sprintf("bucket.%s.created", bucketName), true)
	viper.Set(fmt.Sprintf("bucket.%s.name", bucketName), bucketName)
	if err = viper.WriteConfig(); err != nil {
		log.Println(err)
	}
}

// UploadFile receives a bucket name, a file name and upload it to AWS S3.
func UploadFile(bucketName string, remoteFilename string, localFilename string) error {

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := newAws()
	if err != nil {
		log.Println(err)
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
	log.Printf("file succesfully uploaded to, %s\n", result.Location)
	return nil
}

func DownloadBucket(bucket string, destFolder string) error {

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := newAws()
	if err != nil {
		log.Println(err)
	}

	s3Client := s3.NewFromConfig(awsConfig)

	downloader := manager.NewDownloader(s3.NewFromConfig(awsConfig))

	log.Println("Listing the objects in the bucket:")
	listObjsResponse, err := s3Client.ListObjectsV2(context.Background(),
		&s3.ListObjectsV2Input{
			Bucket: aws.String(bucket),
			Prefix: aws.String(""),
		})

	if err != nil {
		return errors.New("couldn't list bucket contents")
	}

	for _, object := range listObjsResponse.Contents {
		log.Printf("%s (%d bytes, class %v) \n", *object.Key, object.Size, object.StorageClass)

		f, err := pkg.CreateFullPath(filepath.Join(destFolder, *object.Key))
		if err != nil {
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
			return fmt.Errorf("failed to download file, %v", err)
		}
		// close file immediately
		if err = f.Close(); err != nil {
			return err
		}
	}
	return nil
}

func PutTagKubefirstOnBuckets(bucketName string, clusterName string) {

	log.Printf("tagging bucket... %s:%s", bucketName, clusterName)

	s3Client := s3.New(s3.Options{})

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

	_, err := s3Client.PutBucketTagging(context.Background(), input)
	if err != nil {
		// todo: redo it using AWS SDK v2 using SDK types
		//if aerr, ok := err.(awserr.Error); ok {
		//	log.Println(aerr.Error())
		//} else {
		//	log.Println(err.Error())
		//}
		//return
		log.Println(err)
		return
	}
	log.Printf("Bucket: %s tagged successfully", bucketName)
}

func DestroyBucketObjectsAndVersions(bucket, region string) error {

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := newAws()
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		return err
	}

	client := s3.NewFromConfig(awsConfig)

	deleteObject := func(bucket, key, versionId *string) {
		log.Printf("Object: %s/%s\n", *key, aws.ToString(versionId))
		_, err := client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
			Bucket:    bucket,
			Key:       key,
			VersionId: versionId,
		})
		if err != nil {
			log.Printf("Failed to delete object: %v", err)
		}
	}

	in := &s3.ListObjectsV2Input{Bucket: &bucket}
	for {
		out, err := client.ListObjectsV2(context.Background(), in)
		if err != nil {
			log.Printf("Failed to list objects: %v", err)
			return err
		}

		for _, item := range out.Contents {
			deleteObject(&bucket, item.Key, nil)
		}

		if out.IsTruncated {
			in.ContinuationToken = out.ContinuationToken
		} else {
			break
		}
	}

	inVer := &s3.ListObjectVersionsInput{Bucket: &bucket}
	for {
		out, err := client.ListObjectVersions(context.Background(), inVer)
		if err != nil {
			log.Printf("Failed to list version objects: %v", err)
			return err
		}

		for _, item := range out.DeleteMarkers {
			deleteObject(&bucket, item.Key, item.VersionId)
		}

		for _, item := range out.Versions {
			deleteObject(&bucket, item.Key, item.VersionId)
		}

		if out.IsTruncated {
			inVer.VersionIdMarker = out.NextVersionIdMarker
			inVer.KeyMarker = out.NextKeyMarker
		} else {
			break
		}
	}

	_, err = client.DeleteBucket(context.Background(), &s3.DeleteBucketInput{Bucket: &bucket})
	if err != nil {
		log.Printf("Failed to delete bucket: %v", err)
	}
	return nil
}

// DownloadS3File receives a bucket name, filename and download the file at AWS S3
func DownloadS3File(bucketName string, filename string) error {

	// todo: use method approach to avoid new AWS client initializations
	awsConfig, err := newAws()
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

	log.Printf("Downloaded file: %s, file size(bytes): %v", file.Name(), numBytes)

	return nil
}
