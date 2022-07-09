/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/kubefirst/nebulous/internal/gitlab"
	"github.com/kubefirst/nebulous/internal/telemetry"
	"github.com/kubefirst/nebulous/pkg/flare"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
const trackerStage8 = "9 - Send Telemetry"

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

		flare.SetupProgress(9)
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
		infoCmd.Run(cmd, args)
		hostedZoneName, _ := cmd.Flags().GetString("hosted-zone-name")
		metricName := "kubefirst.init.started"
		metricDomain := hostedZoneName
		if !dryrunMode {
			telemetry.SendTelemetry(metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}

		// todo need to check flags and create config

		// hosted zone name:
		// name of the hosted zone to be used for the kubefirst install
		// if suffixed with a dot (eg. kubefirst.com.), the dot will be stripped
		if strings.HasSuffix(hostedZoneName, ".") {
			hostedZoneName = hostedZoneName[:len(hostedZoneName)-1]
		}
		log.Println("hostedZoneName:", hostedZoneName)
		viper.Set("aws.hostedzonename", hostedZoneName)
		viper.Set("argocd.local.service", "http://localhost:8080")
		viper.Set("gitlab.local.service", "http://localhost:8888")
		viper.Set("vault.local.service", "http://localhost:8200")

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

		viper.WriteConfig()

		//! tracker 0
		log.Println("installing kubefirst dependencies")
		download()
		log.Println("dependency installation complete")
		Trackers[trackerStage0].Tracker.Increment(int64(1))

		//! tracker 1
		log.Println("getting aws account information")
		getAccountInfo()
		log.Printf("aws account id: %s\naws user arn: %s", viper.GetString("aws.accountid"), viper.GetString("aws.userarn"))
		Trackers[trackerStage1].Tracker.Increment(int64(1))

		//! tracker 2
		// hosted zone id
		// so we don't have to keep looking it up from the domain name to use it
		hostedZoneId := getDNSInfo(hostedZoneName)
		// viper values set in above function
		log.Println("hostedZoneId:", hostedZoneId)
		Trackers[trackerStage2].Tracker.Increment(int64(1))

		//! tracker 3
		// todo: this doesn't default to testing the dns check
		skipHostedZoneCheck := viper.GetBool("init.hostedzonecheck.enabled")
		if !skipHostedZoneCheck {
			log.Println("skipping hosted zone check")
		} else {
			testHostedZoneLiveness(hostedZoneName, hostedZoneId)
		}
		Trackers[trackerStage3].Tracker.Increment(int64(1))

		//! tracker 4
		//* should we consider going down to a single bucket
		//* for state and artifacts on open source?
		//* hitting a bucket limit on an install might deter someone
		log.Println("creating buckets for state and artifacts")
		bucketRand()
		log.Println("bucket creation complete")
		Trackers[trackerStage4].Tracker.Increment(int64(1))

		//! tracker 5
		log.Println("creating an ssh key pair for your new cloud infrastructure")
		createSshKeyPair()
		log.Println("ssh key pair creation complete")
		Trackers[trackerStage5].Tracker.Increment(int64(1))

		gitopsTemplateGithubOrgOverride := "jarededwards" // discussion point

		log.Printf("cloning and detokenizing the gitops-template repository")
		if gitopsTemplateGithubOrgOverride != "" {
			log.Printf("using --gitops-template-gh-org=%s", gitopsTemplateGithubOrgOverride)
		}
		//! tracker 6
		prepareKubefirstTemplateRepo(gitopsTemplateGithubOrgOverride, "gitops")
		log.Println("clone and detokenization of gitops-template repository complete")
		Trackers[trackerStage6].Tracker.Increment(int64(1))
		//! tracker 7
		log.Printf("cloning and detokenizing the metaphor-template repository")
		prepareKubefirstTemplateRepo("kubefirst", "metaphor")
		log.Println("clone and detokenization of metaphor-template repository complete")
		Trackers[trackerStage7].Tracker.Increment(int64(1))

		metricName = "kubefirst.init.completed"

		if !dryrunMode {
			telemetry.SendTelemetry(metricDomain, metricName)
		} else {
			log.Printf("[#99] Dry-run mode, telemetry skipped:  %s", metricName)
		}

		//! tracker 8
		Trackers[trackerStage8].Tracker.Increment(int64(1))
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
	initCmd.Flags().Bool("clean", false, "delete any local kubefirst content ~/.kubefirst, ~/.flare")

	log.SetPrefix("LOG: ")
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)

	initCmd.PersistentFlags().BoolVarP(&dryrunMode, "dry-run", "s", false, "set to dry-run mode, no changes done on cloud provider selected")
	log.Println("init started")

}

func createSshKeyPair() {
	publicKey := viper.GetString("botpublickey")
	if publicKey == "" {
		publicKey, privateKey, _ := gitlab.GenerateKey()
		viper.Set("botPublicKey", publicKey)
		viper.Set("botPrivateKey", privateKey)
		err := viper.WriteConfig()
		if err != nil {
			log.Panicf("error: could not write to viper config %s", err)
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
