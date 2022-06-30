/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cip8/autoname"
	"github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/kubefirst/nebulous/pkg/flare"
	gitlabSsh "github.com/kubefirst/nebulous/pkg/ssh"
	ssh2 "golang.org/x/crypto/ssh"
)

var Trackers map[string]*flare.ActionTracker

const trackerStage0 = "1 - Load properties"
const trackerStage1 = "2 - Set .flare initial values"
const trackerStage2 = "3 - Test Domain Liveness"
const trackerStage3 = "4 - Create SSH Key Pair"
const trackerStage4 = "5 - Load Templates"
const trackerStage5 = "6 - Download Tools"
const trackerStage6 = "7 - Get Account Info"
const trackerStage7 = "8 - Create Buckets"
const trackerStage8 = "9 - Detokenize"
const trackerStage9 = "10 - Send Telemetry"

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		flare.SetupProgress(10)
		Trackers = make(map[string]*flare.ActionTracker)
		Trackers[trackerStage0] = &flare.ActionTracker{flare.CreateTracker(trackerStage0, int64(1))}
		Trackers[trackerStage1] = &flare.ActionTracker{flare.CreateTracker(trackerStage1, int64(1))}
		Trackers[trackerStage2] = &flare.ActionTracker{flare.CreateTracker(trackerStage2, int64(1))}
		Trackers[trackerStage3] = &flare.ActionTracker{flare.CreateTracker(trackerStage3, int64(1))}
		Trackers[trackerStage4] = &flare.ActionTracker{flare.CreateTracker(trackerStage4, int64(1))}
		Trackers[trackerStage5] = &flare.ActionTracker{flare.CreateTracker(trackerStage5, int64(3))}
		Trackers[trackerStage6] = &flare.ActionTracker{flare.CreateTracker(trackerStage6, int64(1))}
		Trackers[trackerStage7] = &flare.ActionTracker{flare.CreateTracker(trackerStage7, int64(4))}
		Trackers[trackerStage8] = &flare.ActionTracker{flare.CreateTracker(trackerStage8, int64(1))}
		Trackers[trackerStage9] = &flare.ActionTracker{flare.CreateTracker(trackerStage9, int64(1))}
		infoCmd.Run(cmd, args)
		metricName := "kubefirst.init.started"
		metricDomain := "kubefirst.com"
		if !dryrunMode {
			flare.SendTelemetry(metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}

		// todo hack
		awsProfileSet := os.Getenv("AWS_PROFILE")

		if awsProfileSet == "" {
			log.Println("\nhack: !!!!! PLEASE SET AWS PROFILE !!!!!\n\nexport AWS_PROFILE=starter\n")
			os.Exit(1)
		}

		// todo need to check flags and create config

		// hosted zone name:
		// name of the hosted zone to be used for the kubefirst install
		// if suffixed with a dot (eg. kubefirst.com.), the dot will be stripped
		hostedZoneName, _ := cmd.Flags().GetString("hosted-zone-name")
		if strings.HasSuffix(hostedZoneName, ".") {
			hostedZoneName = hostedZoneName[:len(hostedZoneName)-1]
		}
		log.Println("hostedZoneName:", hostedZoneName)
		viper.Set("aws.domainname", hostedZoneName)
		viper.WriteConfig()
		// admin email
		// used for letsencrypt notifications and the gitlab root account
		adminEmail, _ := cmd.Flags().GetString("admin-email")
		log.Println("adminEmail:", adminEmail)
		viper.Set("adminemail", adminEmail)

		// region
		// name of the cloud region to provision resources when resources are region-specific
		region, _ := cmd.Flags().GetString("region")
		viper.Set("aws.region", region)
		log.Println("region:", region)

		// hosted zone id
		// so we don't have to keep looking it up from the domain name to use it
		hostedZoneId := getDNSInfo(hostedZoneName)
		// viper values set in above function
		log.Println("hostedZoneId:", hostedZoneId)
		Trackers[trackerStage0].Tracker.Increment(int64(1))
		Trackers[trackerStage1].Tracker.Increment(int64(1))
		//trackProgress(1, false)
		// todo: this doesn't default to testing the dns check
		if !viper.GetBool("init.hostedzonecheck.enabled") {
			log.Println("skipping hosted zone check")
		} else {
			testHostedZoneLiveness(hostedZoneName, hostedZoneId)
		}
		Trackers[trackerStage2].Tracker.Increment(int64(1))
		// todo generate ssh key --> ~/.kubefirst/ssh-key .pub

		//! step 1
		// todo rm -rf ~/.kubefirst
		// todo make sure - k -n soft-serve port-forward svc/soft-serve 8022:22

		log.Println("calling createSshKeyPair() ")
		createSshKeyPair()
		log.Println("createSshKeyPair() complete\n\n")
		Trackers[trackerStage3].Tracker.Increment(int64(1))

		log.Println("calling cloneGitOpsRepo() function\n")
		cloneGitOpsRepo()
		log.Println("cloneGitOpsRepo() complete\n\n")
		Trackers[trackerStage4].Tracker.Increment(int64(1))

		log.Println("calling download() ")
		download()
		log.Println("download() complete\n\n")

		log.Println("calling getAccountInfo() function\n")
		getAccountInfo()
		log.Println("getAccountInfo() complete\n\n")
		Trackers[trackerStage6].Tracker.Increment(int64(1))

		log.Println("calling bucketRand() function\n")
		bucketRand()
		log.Println("bucketRand() complete\n\n")

		fmt.Println("calling detokenize() ")
		detokenize(fmt.Sprintf("%s/.kubefirst/gitops", home))
		fmt.Println("detokenize() complete\n\n")
		Trackers[trackerStage8].Tracker.Increment(int64(1))

		// modConfigYaml()
		metricName = "kubefirst.init.completed"

		if !dryrunMode {
			flare.SendTelemetry(metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}

		viper.WriteConfig()
		Trackers[trackerStage9].Tracker.Increment(int64(1))
		time.Sleep(time.Millisecond * 100)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().String("hosted-zone-name", "", "the domain to provision the kubefirst platofrm in")
	initCmd.MarkFlagRequired("hosted-zone-name")
	initCmd.Flags().String("admin-email", "", "the email address for the administrator as well as for lets-encrypt certificate emails")
	initCmd.MarkFlagRequired("admin-email")
	initCmd.Flags().String("cloud", "", "the cloud to provision infrastructure in")
	initCmd.MarkFlagRequired("cloud")
	initCmd.Flags().String("region", "", "the region to provision the cloud resources in")
	initCmd.MarkFlagRequired("region")
	initCmd.Flags().Bool("clean", false, "delete any local  kubefirst content ~/.kubefirst, ~/.flare")

	log.SetPrefix("LOG: ")
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)

	initCmd.PersistentFlags().BoolVarP(&dryrunMode, "dry-run", "s", false, "set to dry-run mode, no changes done on cloud provider selected")
	log.Println("init started")

}

func bucketRand() {
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

	buckets := strings.Fields("state-store argo-artifacts gitlab-backup chartmuseum")
	for _, bucket := range buckets {
		bucketExists := viper.GetBool(fmt.Sprintf("bucket.%s.created", bucket))
		if !bucketExists {
			bucketName := fmt.Sprintf("k1-%s-%s", bucket, randomName)

			log.Println("creating", bucket, "bucket", bucketName)

			regionName := viper.GetString("aws.region")
			log.Println("region is ", regionName)
			if !dryrunMode {
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
		Trackers[trackerStage7].Tracker.Increment(int64(1))
	}
}

func getAccountInfo() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Println("failed to load configuration, error:", err)
	}
	// https://aws.github.io/aws-sdk-go-v2/docs/making-requests/#overriding-configuration
	stsClient := sts.NewFromConfig(cfg)
	iamCaller, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		log.Println("oh no error on call", err)
	}

	viper.Set("aws.accountid", *iamCaller.Account)
	viper.Set("aws.userarn", *iamCaller.Arn)
	viper.WriteConfig()
}

func testHostedZoneLiveness(hostedZoneName, hostedZoneId string) {
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
		if !dryrunMode {
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
			// tracker.Message = fmt.Sprintln("dns test", count, "of", 25)
			fmt.Fprintf(os.Stderr, "Could not get record name %s - waiting 10 seconds and trying again: %v\n", route53RecordName, err)
			time.Sleep(10 * time.Second)
		} else {
			for _, ip := range ips {
				// todo check ip against route53RecordValue in some capacity so we can pivot the value for testing
				log.Printf("%s. in TXT record value: %s\n", route53RecordName, ip)
				//tracker.MarkAsDone()
				count = 26
			}
		}
		if count == 25 {
			log.Println("unable to resolve hosted zone dns record. please check your domain registrar")
			//tracker.MarkAsErrored()
			//pw.Stop()
			os.Exit(1)
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
	// 	fmt.Println("error deleting route 53 record after liveness test")
	// }
	// fmt.Println("record deletion status is ", *&recordDelete.ChangeInfo.Status)

}

func modConfigYaml() {

	file, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		log.Println("error reading file", err)
	}

	newFile := strings.Replace(string(file), "allow-keyless: false", "allow-keyless: true", -1)

	err = ioutil.WriteFile("./config.yaml", []byte(newFile), 0)
	if err != nil {
		panic(err)
	}
}

func getDNSInfo(hostedZoneName string) string {

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

	var zoneId string

	for _, zone := range hostedZones.HostedZones {
		if *zone.Name == fmt.Sprintf(`%s%s`, hostedZoneName, ".") {
			zoneId = returnHostedZoneId(*zone.Id)
			log.Printf(`found entry for user submitted domain %s, using hosted zone id %s`, hostedZoneName, zoneId)
			viper.Set("aws.domainname", hostedZoneName)
			viper.Set("aws.domainid", zoneId)
			viper.WriteConfig()
		}
	}
	return zoneId

}

func returnHostedZoneId(rawZoneId string) string {
	return strings.Split(rawZoneId, "/")[2]
}

func publicKey() (*ssh.PublicKeys, error) {
	var publicKey *ssh.PublicKeys
	publicKey, err := ssh.NewPublicKeys("git", []byte(viper.GetString("botprivatekey")), "")
	if err != nil {
		return nil, err
	}
	return publicKey, err
}

func cloneGitOpsRepo() {

	url := "https://github.com/kubefirst/gitops-template"
	directory := fmt.Sprintf("%s/.kubefirst/gitops", home)

	// Clone the given repository to the given directory
	log.Println("git clone", url, directory)

	_, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		log.Println(err)
	}

	println("downloaded gitops repo from template to directory", home, "/.kubefirst/gitops")
}

func configureSoftServe() {
	// todo clone repo
	// todo manipulate config.yaml
	// todo git add / commit / push
	url := "ssh://127.0.0.1:8022/config"
	directory := fmt.Sprintf("%s/.kubefirst/config", home)

	// Clone the given repository to the given directory
	log.Println("git clone", url, directory)

	auth, _ := publicKey()

	auth.HostKeyCallback = ssh2.InsecureIgnoreHostKey()

	repo, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL:  url,
		Auth: auth,
	})
	if err != nil {
		log.Println("error!, ", err)
	}

	file, err := ioutil.ReadFile(fmt.Sprintf("%s/config.yaml", directory))
	if err != nil {
		log.Println("error reading file", err)
	}

	newFile := strings.Replace(string(file), "allow-keyless: false", "allow-keyless: true", -1)

	err = ioutil.WriteFile(fmt.Sprintf("%s/config.yaml", directory), []byte(newFile), 0)
	if err != nil {
		panic(err)
	}

	println("re-wrote config.yaml", home, "/.kubefirst/config")

	w, _ := repo.Worktree()

	log.Println("Committing new changes...")
	w.Add(".")
	w.Commit("updating soft-serve server config", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: installerEmail,
			When:  time.Now(),
		},
	})

	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if err != nil {
		log.Println("error pushing to remote", err)
	}

}

func pushGitopsToSoftServe() {

	directory := fmt.Sprintf("%s/.kubefirst/gitops", home)

	// // Clone the given repository to the given directory
	log.Println("open %s git repo", directory)

	repo, err := git.PlainOpen(directory)
	if err != nil {
		log.Println("error opening the directory ", directory, err)
	}

	log.Println("git remote add origin ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops")
	_, err = repo.CreateRemote(&gitConfig.RemoteConfig{
		Name: "soft",
		URLs: []string{"ssh://127.0.0.1:8022/gitops"},
	})
	if err != nil {
		log.Println("Error creating remote repo:", err)
		os.Exit(1)
	}
	w, _ := repo.Worktree()

	log.Println("Committing new changes...")
	w.Add(".")
	w.Commit("setting new remote upstream to soft-serve", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: installerEmail,
			When:  time.Now(),
		},
	})

	auth, _ := publicKey()

	auth.HostKeyCallback = ssh2.InsecureIgnoreHostKey()

	err = repo.Push(&git.PushOptions{
		RemoteName: "soft",
		Auth:       auth,
	})
	if err != nil {
		log.Println("error pushing to remote", err)
	}

}

func pushGitopsToGitLab() {
	domain := viper.GetString("aws.domainname")

	detokenize(fmt.Sprintf("%s/.kubefirst/gitops", home))
	directory := fmt.Sprintf("%s/.kubefirst/gitops", home)

	repo, err := git.PlainOpen(directory)
	if err != nil {
		log.Println("error opening the directory ", directory, err)
	}

	//upstream := fmt.Sprintf("ssh://gitlab.%s:22:kubefirst/gitops", viper.GetString("aws.domainname"))
	// upstream := "git@gitlab.kube1st.com:kubefirst/gitops.git"
	upstream := fmt.Sprintf("https://gitlab.%s/kubefirst/gitops.git", domain)
	log.Println("git remote add gitlab at url", upstream)

	_, err = repo.CreateRemote(&gitConfig.RemoteConfig{
		Name: "gitlab",
		URLs: []string{upstream},
	})
	if err != nil {
		log.Println("Error creating remote repo:", err)
	}
	w, _ := repo.Worktree()

	os.RemoveAll(directory + "/terraform/base/.terraform")
	os.RemoveAll(directory + "/terraform/gitlab/.terraform")
	os.RemoveAll(directory + "/terraform/vault/.terraform")

	log.Println("Committing new changes...")
	w.Add(".")
	_, err = w.Commit("setting new remote upstream to gitlab", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: installerEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		log.Println("error committing changes", err)
	}

	log.Println("setting auth...")
	// auth, _ := publicKey()
	// auth.HostKeyCallback = ssh2.InsecureIgnoreHostKey()

	auth := &gitHttp.BasicAuth{
		Username: "root",
		Password: viper.GetString("gitlab.token"),
	}

	err = repo.Push(&git.PushOptions{
		RemoteName: "gitlab",
		Auth:       auth,
	})
	if err != nil {
		log.Println("error pushing to remote", err)
	}

}

func detokenize(path string) {

	err := filepath.Walk(path, detokenizeDirectory)
	if err != nil {
		panic(err)
	}
}

func detokenizeDirectory(path string, fi os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if fi.IsDir() {
		return nil //
	}

	if strings.Contains(path, ".git") || strings.Contains(path, ".terraform") {
		return nil
	}

	matched, err := filepath.Match("*", fi.Name())

	if err != nil {
		panic(err)
	}

	if matched {
		read, err := ioutil.ReadFile(path)
		if err != nil {
			panic(err)
		}
		// todo should detokenize be a switch statement based on a value found in viper?
		gitlabConfigured := viper.GetBool("gitlab.keyuploaded")

		newContents := ""

		if gitlabConfigured {
			newContents = strings.Replace(string(read), "ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops", fmt.Sprintf("https://gitlab.%s/kubefirst/gitops.git", viper.GetString("aws.domainname")), -1)
		} else {
			newContents = strings.Replace(string(read), "https://gitlab.<AWS_HOSTED_ZONE_NAME>/kubefirst/gitops.git", "ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops", -1)
		}

		botPublicKey := viper.GetString("botpublickey")
		domainId := viper.GetString("aws.domainid")
		domainName := viper.GetString("aws.domainname")
		bucketStateStore := viper.GetString("bucket.state-store.name")
		bucketArgoArtifacts := viper.GetString("bucket.argo-artifacts.name")
		bucketGitlabBackup := viper.GetString("bucket.gitlab-backup.name")
		bucketChartmuseum := viper.GetString("bucket.chartmuseum.name")
		region := viper.GetString("aws.region")
		adminEmail := viper.GetString("adminemail")
		awsAccountId := viper.GetString("aws.accountid")
		kmsKeyId := viper.GetString("vault.kmskeyid")

		newContents = strings.Replace(newContents, "<SOFT_SERVE_INITIAL_ADMIN_PUBLIC_KEY>", strings.TrimSpace(botPublicKey), -1)
		newContents = strings.Replace(newContents, "<TF_STATE_BUCKET>", bucketStateStore, -1)
		newContents = strings.Replace(newContents, "<ARGO_ARTIFACT_BUCKET>", bucketArgoArtifacts, -1)
		newContents = strings.Replace(newContents, "<GITLAB_BACKUP_BUCKET>", bucketGitlabBackup, -1)
		newContents = strings.Replace(newContents, "<CHARTMUSEUM_BUCKET>", bucketChartmuseum, -1)
		newContents = strings.Replace(newContents, "<AWS_HOSTED_ZONE_ID>", domainId, -1)
		newContents = strings.Replace(newContents, "<AWS_HOSTED_ZONE_NAME>", domainName, -1)
		newContents = strings.Replace(newContents, "<AWS_DEFAULT_REGION>", region, -1)
		newContents = strings.Replace(newContents, "<EMAIL_ADDRESS>", adminEmail, -1)
		newContents = strings.Replace(newContents, "<AWS_ACCOUNT_ID>", awsAccountId, -1)
		if kmsKeyId != "" {
			newContents = strings.Replace(newContents, "<KMS_KEY_ID>", kmsKeyId, -1)
		}

		if viper.GetBool("create.terraformapplied.gitlab") {
			newContents = strings.Replace(newContents, "<AWS_HOSTED_ZONE_NAME>", domainName, -1)
			newContents = strings.Replace(newContents, "<AWS_DEFAULT_REGION>", region, -1)
			newContents = strings.Replace(newContents, "<AWS_ACCOUNT_ID>", awsAccountId, -1)
		}

		err = ioutil.WriteFile(path, []byte(newContents), 0)
		if err != nil {
			panic(err)
		}

	}

	return nil
}

func download() {
	toolsDir := fmt.Sprintf("%s/.kubefirst/tools", home)

	err := os.Mkdir(toolsDir, 0777)
	if err != nil {
		log.Println("error creating directory %s", toolsDir, err)
	}

	kubectlVersion := "v1.20.0"
	kubectlDownloadUrl := fmt.Sprintf("https://dl.k8s.io/release/%s/bin/%s/%s/kubectl", kubectlVersion, localOs, localArchitecture)
	downloadFile(kubectlClientPath, kubectlDownloadUrl)
	os.Chmod(kubectlClientPath, 0755)

	// todo this kubeconfig is not available to us until we have run the terraform in base/
	os.Setenv("KUBECONFIG", kubeconfigPath)
	log.Println("going to print the kubeconfig env in runtime", os.Getenv("KUBECONFIG"))

	kubectlVersionCmd := exec.Command(kubectlClientPath, "version", "--client", "--short")
	kubectlVersionCmd.Stdout = os.Stdout
	kubectlVersionCmd.Stderr = os.Stderr
	err = kubectlVersionCmd.Run()
	if err != nil {
		log.Println("failed to call kubectlVersionCmd.Run(): %v", err)
	}
	Trackers[trackerStage5].Tracker.Increment(int64(1))
	// argocdVersion := "v2.3.4"
	// argocdDownloadUrl := fmt.Sprintf("https://github.com/argoproj/argo-cd/releases/download/%s/argocd-%s-%s", argocdVersion, localOs, localArchitecture)
	// argocdClientPath := fmt.Sprintf("%s/.kubefirst/tools/argocd", home)
	// downloadFile(argocdClientPath, argocdDownloadUrl)
	// os.Chmod(argocdClientPath, 755)

	// argocdVersionCmd := exec.Command(argocdClientPath, "version", "--client", "--short")
	// argocdVersionCmd.Stdout = os.Stdout
	// argocdVersionCmd.Stderr = os.Stderr
	// err = argocdVersionCmd.Run()
	// if err != nil {
	// 	fmt.Println("failed to call argocdVersionCmd.Run(): %v", err)
	// }

	// todo adopt latest helmVersion := "v3.9.0"
	terraformVersion := "1.0.11"
	// terraformClientPath := fmt.Sprintf("./%s-%s/terraform", localOs, localArchitecture)
	terraformDownloadUrl := fmt.Sprintf("https://releases.hashicorp.com/terraform/%s/terraform_%s_%s_%s.zip", terraformVersion, terraformVersion, localOs, localArchitecture)
	terraformDownloadZipPath := fmt.Sprintf("%s/.kubefirst/tools/terraform.zip", home)
	downloadFile(terraformDownloadZipPath, terraformDownloadUrl)
	// terraformZipDownload, err := os.Open(terraformDownloadZipPath)
	if err != nil {
		log.Println("error reading terraform file")
	}
	unzipDirectory := fmt.Sprintf("%s/.kubefirst/tools", home)
	unzip(terraformDownloadZipPath, unzipDirectory)

	os.Chmod(unzipDirectory, 0777)
	os.Chmod(fmt.Sprintf("%s/terraform", unzipDirectory), 0755)
	Trackers[trackerStage5].Tracker.Increment(int64(1))

	// todo adopt latest helmVersion := "v3.9.0"
	helmVersion := "v3.2.1"
	helmDownloadUrl := fmt.Sprintf("https://get.helm.sh/helm-%s-%s-%s.tar.gz", helmVersion, localOs, localArchitecture)
	helmDownloadTarGzPath := fmt.Sprintf("%s/.kubefirst/tools/helm.tar.gz", home)
	downloadFile(helmDownloadTarGzPath, helmDownloadUrl)
	helmTarDownload, err := os.Open(helmDownloadTarGzPath)
	if err != nil {
		log.Println("error reading helm file")
	}
	extractFileFromTarGz(helmTarDownload, fmt.Sprintf("%s-%s/helm", localOs, localArchitecture), helmClientPath)
	os.Chmod(helmClientPath, 0755)
	helmVersionCmd := exec.Command(helmClientPath, "version", "--client", "--short")

	// currently argocd init values is generated by flare nebulous ssh

	// todo helm install argocd --create-namespace --wait --values ~/.kubefirst/argocd-init-values.yaml argo/argo-cd
	helmVersionCmd.Stdout = os.Stdout
	helmVersionCmd.Stderr = os.Stderr
	err = helmVersionCmd.Run()
	if err != nil {
		log.Println("failed to call helmVersionCmd.Run(): %v", err)
	}
	Trackers[trackerStage5].Tracker.Increment(int64(1))

}

func downloadFile(filepath string, url string) (err error) {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
func extractFileFromTarGz(gzipStream io.Reader, tarAddress string, targetFilePath string) {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		log.Fatal("extractTarGz: NewReader failed")
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("extractTarGz: Next() failed: %s", err.Error())
		}
		log.Println(header.Name)
		if header.Name == tarAddress {
			switch header.Typeflag {
			case tar.TypeReg:
				outFile, err := os.Create(targetFilePath)
				if err != nil {
					log.Println("extractTarGz: Create() failed: %s", err.Error())
				}
				if _, err := io.Copy(outFile, tarReader); err != nil {
					log.Println("extractTarGz: Copy() failed: %s", err.Error())
				}
				outFile.Close()

			default:
				log.Println(
					"extractTarGz: uknown type: %s in %s",
					header.Typeflag,
					header.Name)
			}

		}
	}
}

func extractTarGz(gzipStream io.Reader) {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		log.Fatal("extractTarGz: NewReader failed")
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Println("extractTarGz: Next() failed: %s", err.Error())
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(header.Name, 0755); err != nil {
				log.Println("extractTarGz: Mkdir() failed: %s", err.Error())
			}
		case tar.TypeReg:
			outFile, err := os.Create(header.Name)
			if err != nil {
				log.Println("extractTarGz: Create() failed: %s", err.Error())
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				log.Println("extractTarGz: Copy() failed: %s", err.Error())
			}
			outFile.Close()

		default:
			log.Println(
				"extractTarGz: uknown type: %s in %s",
				header.Typeflag,
				header.Name)
		}

	}
}

func createSoftServe(kubeconfigPath string) {

	toolsDir := fmt.Sprintf("%s/.kubefirst/tools", home)

	err := os.Mkdir(toolsDir, 0777)
	if err != nil {
		log.Println("error creating directory %s", toolsDir, err)
	}

	// create soft-serve stateful set
	softServePath := fmt.Sprintf("%s/.kubefirst/gitops/components/soft-serve/manifests.yaml", home)
	kubectlCreateSoftServeCmd := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "soft-serve", "apply", "-f", softServePath, "--wait")
	kubectlCreateSoftServeCmd.Stdout = os.Stdout
	kubectlCreateSoftServeCmd.Stderr = os.Stderr
	err = kubectlCreateSoftServeCmd.Run()
	if err != nil {
		log.Println("failed to call kubectlCreateSoftServeCmd.Run(): %v", err)
	}
}

func helmInstallArgocd(home string, kubeconfigPath string) {

	argocdCreated := viper.GetBool("create.argocd.helm")
	if !argocdCreated {
		helmClientPath := fmt.Sprintf("%s/.kubefirst/tools/helm", home)

		// ! commenting out until a clean execution is necessary // create namespace
		helmRepoAddArgocd := exec.Command(helmClientPath, "--kubeconfig", kubeconfigPath, "repo", "add", "argo", "https://argoproj.github.io/argo-helm")
		helmRepoAddArgocd.Stdout = os.Stdout
		helmRepoAddArgocd.Stderr = os.Stderr
		err := helmRepoAddArgocd.Run()
		if err != nil {
			log.Println("failed to call helmRepoAddArgocd.Run(): %v", err)
		}

		helmRepoUpdate := exec.Command(helmClientPath, "--kubeconfig", kubeconfigPath, "repo", "update")
		helmRepoUpdate.Stdout = os.Stdout
		helmRepoUpdate.Stderr = os.Stderr
		err = helmRepoUpdate.Run()
		if err != nil {
			log.Println("failed to call helmRepoUpdate.Run(): %v", err)
		}

		helmInstallArgocdCmd := exec.Command(helmClientPath, "--kubeconfig", kubeconfigPath, "upgrade", "--install", "argocd", "--namespace", "argocd", "--create-namespace", "--wait", "--values", fmt.Sprintf("%s/.kubefirst/argocd-init-values.yaml", home), "argo/argo-cd")
		helmInstallArgocdCmd.Stdout = os.Stdout
		helmInstallArgocdCmd.Stderr = os.Stderr
		err = helmInstallArgocdCmd.Run()
		if err != nil {
			log.Println("failed to call helmInstallArgocdCmd.Run(): %v", err)
		}

		viper.Set("create.argocd.helm", true)
		err = viper.WriteConfig()
		if err != nil {
			log.Println(err)
		}
	}
}

func createSshKeyPair() {
	publicKey := viper.GetString("botpublickey")
	if publicKey == "" {
		log.Println("generating new key pair")
		publicKey, privateKey, _ := gitlabSsh.GenerateKey()
		viper.Set("botPublicKey", publicKey)
		viper.Set("botPrivateKey", privateKey)
		err := viper.WriteConfig()
		if err != nil {
			log.Println(err)
		}
	}
	publicKey = viper.GetString("botpublickey")
	privateKey := viper.GetString("botprivatekey")

	var argocdInitValuesYaml = []byte(fmt.Sprintf(`
server:
  additionalApplications:
  - name: registry
    namespace: argocd
    additionalLabels: {}
    additionalAnnotations: {}
    finalizers:
    - resources-finalizer.argocd.argoproj.io
    project: default
    source:
      repoURL: ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops
      targetRevision: HEAD
      path: registry
    destination:
      server: https://kubernetes.default.svc
      namespace: argocd
    syncPolicy:
      automated:
        prune: true
        selfHeal: true
      syncOptions:
      - CreateNamespace=true
configs:
  repositories:
    soft-serve-gitops:
      url: ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops
      insecure: 'true'
      type: git
      name: soft-serve-gitops
  credentialTemplates:
    ssh-creds:
      url: ssh://soft-serve.soft-serve.svc.cluster.local:22
      sshPrivateKey: |
        %s
`, strings.ReplaceAll(privateKey, "\n", "\n        ")))

	// fmt.Println("argo init vals:\n", string(argocdInitValuesYaml))

	err := ioutil.WriteFile(fmt.Sprintf("%s/.kubefirst/argocd-init-values.yaml", home), argocdInitValuesYaml, 0644)
	if err != nil {
		log.Println("received an error while writing the argocd-init-values.yaml file", err.Error())
		panic("error: argocd-init-values.yaml" + err.Error())
	}
}

func unzip(zipFilepath string, unzipDirectory string) {
	dst := unzipDirectory
	archive, err := zip.OpenReader(zipFilepath)
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(dst, f.Name)
		log.Println("unzipping file ", filePath)

		if !strings.HasPrefix(filePath, filepath.Clean(dst)+string(os.PathSeparator)) {
			log.Println("invalid file path")
			return
		}
		if f.FileInfo().IsDir() {
			log.Println("creating directory...")
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			panic(err)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		dstFile.Close()
		fileInArchive.Close()
	}
}
