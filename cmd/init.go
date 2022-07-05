/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/kubefirst/nebulous/pkg/flare"
	gitlabSsh "github.com/kubefirst/nebulous/pkg/ssh"	
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

		log.Println("calling detokenize() ")
		detokenize(fmt.Sprintf("%s/.kubefirst/gitops", home))
		log.Println("detokenize() complete\n\n")
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


















func createSshKeyPair() {
	publicKey := viper.GetString("botpublickey")
	if publicKey == "" {
		log.Println("generating new key pair")
		publicKey, privateKey, _ := gitlabSsh.GenerateKey()
		viper.Set("botPublicKey", publicKey)
		viper.Set("botPrivateKey", privateKey)
		err := viper.WriteConfig()
		if err != nil {
			log.Panicf("error: could not write to viper config")
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

	err := ioutil.WriteFile(fmt.Sprintf("%s/.kubefirst/argocd-init-values.yaml", home), argocdInitValuesYaml, 0644)
	if err != nil {
		log.Panicf("error: could not write argocd-init-values.yaml %s", err)
	}
}


