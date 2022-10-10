package ciTools

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/repo"
	"github.com/spf13/viper"
)

// DeployOnGitlab - Deploy CI applications on gitlab install
func DeployOnGitlab(globalFlags flagset.GlobalFlags, bucketName string) error {
	config := configs.ReadConfig()
	if globalFlags.DryRun {
		log.Printf("[#99] Dry-run mode, DeployOnGitlab skipped.")
		return nil
	}
	log.Printf("cloning and detokenizing the ci-template repository")
	repo.PrepareKubefirstTemplateRepo(globalFlags.DryRun, config, viper.GetString("gitops.owner"), "ci", viper.GetString("ci.branch"), viper.GetString("template.tag"))
	log.Println("clone and detokenization of ci-template repository complete")

	err := SedBucketName("<BUCKET_NAME>", bucketName)
	if err != nil {
		log.Panicf("Error sed bucket name on CI repository: %s", err)
		return err
	}

	ciLocation := fmt.Sprintf("%s/ci/components/argo-gitlab/ci.yaml", config.K1FolderPath)

	err = DetokenizeCI("<CI_CLUSTER_NAME>", viper.GetString("ci.cluster.name"), ciLocation)
	if err != nil {
		log.Println(err)
	}
	err = DetokenizeCI("<CI_S3_SUFFIX>", viper.GetString("ci.s3.suffix"), ciLocation)
	if err != nil {
		log.Println(err)
	}
	err = DetokenizeCI("<CI_HOSTED_ZONE_NAME>", viper.GetString("ci.hosted.zone.name"), ciLocation)
	if err != nil {
		log.Println(err)
	}

	if !viper.GetBool("gitlab.ci-pushed") {
		log.Println("Pushing ci repo to origin gitlab")
		gitlab.PushGitRepo(globalFlags.DryRun, config, "gitlab", "ci")
		viper.Set("gitlab.ci-pushed", true)
		viper.WriteConfig()
		log.Println("clone and detokenization of ci-template repository complete")
	}

	return nil
}

func SedBucketName(old, new string) error {
	cfg := configs.ReadConfig()
	providerFile := fmt.Sprintf("%s/ci/terraform/base/provider.tf", cfg.K1FolderPath)

	fileData, err := ioutil.ReadFile(providerFile)
	if err != nil {
		return err
	}

	fileString := string(fileData)
	fileString = strings.ReplaceAll(fileString, old, new)
	fileData = []byte(fileString)

	err = ioutil.WriteFile(providerFile, fileData, 0o600)
	if err != nil {
		return err
	}

	return nil
}

func DetokenizeCI(old, new, ciLocation string) error {
	ciFile := ciLocation

	fileData, err := ioutil.ReadFile(ciFile)
	if err != nil {
		return err
	}

	fileString := string(fileData)
	fileString = strings.ReplaceAll(fileString, old, new)
	fileData = []byte(fileString)

	err = ioutil.WriteFile(ciFile, fileData, 0o600)
	if err != nil {
		return err
	}
	return nil
}
