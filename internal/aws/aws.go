package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cip8/autoname"
	"github.com/kubefirst/nebulous/pkg"
	"github.com/spf13/viper"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func BucketRand(dryRun bool, trackers map[string]*pkg.ActionTracker) {

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(viper.GetString("aws.region"))},
	)
	if err != nil {
		log.Println("failed to attempt bucket creation ", err.Error())
		os.Exit(1)
	}

	s3Client := s3.New(sess)

	randomName := strings.ReplaceAll(autoname.Generate(), "_", "-")
	viper.Set("bucket.rand", randomName)

	trackers[pkg.TrackerStage7].Tracker.Increment(int64(1))

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
					_, err = s3Client.CreateBucket(&s3.CreateBucketInput{
						Bucket: &bucketName,
					})
				} else {
					_, err = s3Client.CreateBucket(&s3.CreateBucketInput{
						Bucket: &bucketName,
						CreateBucketConfiguration: &s3.CreateBucketConfiguration{
							LocationConstraint: aws.String(regionName),
						},
					})
				}
				if err != nil {
					log.Println("failed to create bucket "+bucketName, err.Error())
					os.Exit(1)
				}
			} else {
				log.Printf("[#99] Dry-run mode, bucket creation skipped:  %s", bucketName)
			}
			viper.Set(fmt.Sprintf("bucket.%s.created", bucket), true)
			viper.Set(fmt.Sprintf("bucket.%s.name", bucket), bucketName)
			viper.WriteConfig()
		}
		log.Printf("bucket %s exists", viper.GetString(fmt.Sprintf("bucket.%s.name", bucket)))
	}
}

func GetAccountInfo() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Panicf("failed to load configuration, error: %s", err)
	}
	stsClient := sts.NewFromConfig(cfg)
	iamCaller, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		log.Panicf("error: could not get caller identity %s", err)
	}

	viper.Set("aws.accountid", *iamCaller.Account)
	viper.Set("aws.userarn", *iamCaller.Arn)
	viper.WriteConfig()
}

func TestHostedZoneLiveness(dryRun bool, hostedZoneName, hostedZoneId string) {
	//tracker := progress.Tracker{Message: "testing hosted zone", Total: 25}

	// todo need to create single client and pass it
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Println("failed to load configuration, error:", err)
	}
	// https://aws.github.io/aws-sdk-go-v2/docs/making-requests/#overriding-configuration
	route53Client := route53.NewFromConfig(cfg)

	// todo when checking to see if hosted zone exists print ns records for user to verity in dns registrar
	route53RecordName := fmt.Sprintf("kubefirst-liveness.%s", hostedZoneName)
	route53RecordValue := "domain record propagated"

	log.Println("checking to see if record", route53RecordName, "exists")
	log.Println("hostedZoneId", hostedZoneId)
	log.Println("route53RecordName", route53RecordName)

	recordList, err := route53Client.ListResourceRecordSets(context.TODO(), &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(hostedZoneId),
		StartRecordName: aws.String(route53RecordName),
		StartRecordType: "TXT",
	})
	if err != nil {
		log.Println("failed read route53 ", err.Error())
		os.Exit(1)
	}

	if len(recordList.ResourceRecordSets) == 0 {
		if !dryRun {
			record, err := route53Client.ChangeResourceRecordSets(context.TODO(), &route53.ChangeResourceRecordSetsInput{
				ChangeBatch: &types.ChangeBatch{
					Changes: []types.Change{
						{
							Action: "CREATE",
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
			}
			log.Println("record creation status is ", record.ChangeInfo.Status)
		} else {
			log.Printf("[#99] Dry-run mode, route53 creation/update skipped:  %s", route53RecordName)
		}
	}
	count := 0
	// todo need to exit after n number of minutes and tell them to check ns records
	// todo this logic sucks
	for count <= 25 {
		count++
		//tracker.Increment(1)
		//log.Println(text.Faint.Sprintf("[INFO] dns test %d of 25", count))

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
				//tracker.MarkAsDone()
				count = 26
			}
		}
		if count == 25 {
			//tracker.MarkAsErrored()
			//pw.Stop()
			log.Panicf("unable to resolve hosted zone dns record. please check your domain registrar")
		}
	}
	// todo delete route53 record

	// recordDelete, err := route53Client.ChangeResourceRecordSets(context.TODO(), &route53.ChangeResourceRecordSetsInput{
	// 	ChangeBatch: &types.ChangeBatch{
	// 		Changes: []types.Change{
	// 			{
	// 				Action: "DELETE",
	// 				ResourceRecordSet: &types.ResourceRecordSet{
	// 					Name: aws.String(route53RecordName),
	// 					Type: "A",
	// 					ResourceRecords: []types.ResourceRecord{
	// 						{
	// 							Value: aws.String(route53RecordValue),
	// 						},
	// 					},
	// 					TTL:           aws.Int64(10),
	// 					Weight:        aws.Int64(100),
	// 					SetIdentifier: aws.String("CREATE sanity check for kubefirst installation"),
	// 				},
	// 			},
	// 		},
	// 		Comment: aws.String("CREATE sanity check dns record."),
	// 	},
	// 	HostedZoneId: aws.String(hostedZoneId),
	// })
	// if err != nil {
	// 	log.Println("error deleting route 53 record after liveness test")
	// }
	// log.Println("record deletion status is ", *&recordDelete.ChangeInfo.Status)

}

func GetDNSInfo(hostedZoneName string) string {

	log.Println("GetDNSInfo (working...)")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Println("failed to load configuration, error:", err)
	}
	// https://aws.github.io/aws-sdk-go-v2/docs/making-requests/#overriding-configuration
	route53Client := route53.NewFromConfig(cfg)
	hostedZones, err := route53Client.ListHostedZonesByName(context.TODO(), &route53.ListHostedZonesByNameInput{
		DNSName: &hostedZoneName,
	})
	if err != nil {
		log.Println("oh no error on call", err)
	}

	var hostedZoneId string

	for _, zone := range hostedZones.HostedZones {
		if *zone.Name == fmt.Sprintf(`%s%s`, hostedZoneName, ".") {
			hostedZoneId = returnHostedZoneId(*zone.Id)
			log.Printf(`found entry for user submitted domain %s, using hosted zone id %s`, hostedZoneName, hostedZoneId)
			viper.Set("aws.hostedzonename", hostedZoneName)
			viper.Set("aws.hostedzoneid", hostedZoneId)
			viper.WriteConfig()
		}
	}
	log.Println("GetDNSInfo (done)")
	return hostedZoneId

}

func ReturnHostedZoneId(rawZoneId string) string {
	return strings.Split(rawZoneId, "/")[2]
}

func ListBucketsInUse() []string {
	//Read flare file
	//Iterate over buckets
	//check if bucket exist
	//buckets := make([]map[string]string, 0)
	//var m map[string]string
	var bucketsInUse []string
	bucketsConfig := viper.AllKeys()
	for _, bucketKey := range bucketsConfig {
		match := strings.HasPrefix(bucketKey, "bucket.") && strings.HasSuffix(bucketKey, ".name")
		if match {
			bucketName := viper.GetString(bucketKey)
			bucketsInUse = append(bucketsInUse, bucketName)
		}
	}
	return bucketsInUse
}

func DestroyBucket(bucketName string) {

	s3Client := s3.New(GetAWSSession())

	log.Printf("Attempt to delete: %s", bucketName)
	_, errHead := s3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: &bucketName,
	})
	if errHead != nil {
		if aerr, ok := errHead.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				log.Println("Bucket Error:", s3.ErrCodeNoSuchBucket, aerr.Error())
			default:
				log.Println("Bucket Error:", aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(errHead.Error())
		}
	} else {
		//if exist, we can delete it
		_, err := s3Client.DeleteBucket(&s3.DeleteBucketInput{
			Bucket: &bucketName,
		})
		if err != nil {
			log.Panicf("failed to delete bucket "+bucketName, err.Error())
		}

	}

}

func GetAWSSession() *session.Session {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(viper.GetString("aws.region"))},
	)
	if err != nil {
		log.Panicf("failed to get session ", err.Error())
	}
	return sess
}

func DestroyBucketsInUse(destroyBuckets bool) {
	if destroyBuckets {
		log.Println("Execute: DestroyBucketsInUse")
		for _, bucket := range ListBucketsInUse() {
			DestroyBucket(bucket)
		}
	} else {
		log.Println("Skip: DestroyBucketsInUse")
	}
}
