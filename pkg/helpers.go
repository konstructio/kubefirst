package pkg

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kubefirst/kubefirst/configs"

	"github.com/spf13/viper"
)

func Detokenize(path string) {

	err := filepath.Walk(path, DetokenizeDirectory)
	if err != nil {
		log.Panic(err)
	}
}

func DetokenizeDirectory(path string, fi os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if fi.IsDir() {
		return nil //
	}

	if strings.Contains(path, ".gitClient") || strings.Contains(path, ".terraform") || strings.Contains(path, ".git/") {
		return nil
	}

	matched, err := filepath.Match("*", fi.Name())

	if err != nil {
		log.Panic(err)
	}

	if matched {
		read, err := ioutil.ReadFile(path)
		if err != nil {
			log.Panic(err)
		}
		// todo should Detokenize be a switch statement based on a value found in viper?
		gitlabConfigured := viper.GetBool("gitlab.keyuploaded")

		newContents := ""

		if gitlabConfigured {
			newContents = strings.Replace(string(read), "ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops", fmt.Sprintf("https://gitlab.%s/kubefirst/gitops.git", viper.GetString("aws.hostedzonename")), -1)
		} else {
			newContents = strings.Replace(string(read), "https://gitlab.<AWS_HOSTED_ZONE_NAME>/kubefirst/gitops.git", "ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops", -1)
		}

		botPublicKey := viper.GetString("botpublickey")
		hostedZoneId := viper.GetString("aws.hostedzoneid")
		hostedZoneName := viper.GetString("aws.hostedzonename")
		bucketStateStore := viper.GetString("bucket.state-store.name")
		bucketArgoArtifacts := viper.GetString("bucket.argo-artifacts.name")
		bucketGitlabBackup := viper.GetString("bucket.gitlab-backup.name")
		bucketChartmuseum := viper.GetString("bucket.chartmuseum.name")
		region := viper.GetString("aws.region")
		adminEmail := viper.GetString("adminemail")
		awsAccountId := viper.GetString("aws.accountid")
		kmsKeyId := viper.GetString("vault.kmskeyid")
		clusterName := viper.GetString("cluster-name")
		argocdOidcClientId := viper.GetString(("gitlab.oidc.argocd.applicationid"))

		newContents = strings.Replace(newContents, "<SOFT_SERVE_INITIAL_ADMIN_PUBLIC_KEY>", strings.TrimSpace(botPublicKey), -1)
		newContents = strings.Replace(newContents, "<TF_STATE_BUCKET>", bucketStateStore, -1)
		newContents = strings.Replace(newContents, "<ARGO_ARTIFACT_BUCKET>", bucketArgoArtifacts, -1)
		newContents = strings.Replace(newContents, "<GITLAB_BACKUP_BUCKET>", bucketGitlabBackup, -1)
		newContents = strings.Replace(newContents, "<CHARTMUSEUM_BUCKET>", bucketChartmuseum, -1)
		newContents = strings.Replace(newContents, "<AWS_HOSTED_ZONE_ID>", hostedZoneId, -1)
		newContents = strings.Replace(newContents, "<AWS_HOSTED_ZONE_NAME>", hostedZoneName, -1)
		newContents = strings.Replace(newContents, "<AWS_DEFAULT_REGION>", region, -1)
		newContents = strings.Replace(newContents, "<EMAIL_ADDRESS>", adminEmail, -1)
		newContents = strings.Replace(newContents, "<AWS_ACCOUNT_ID>", awsAccountId, -1)
		if kmsKeyId != "" {
			newContents = strings.Replace(newContents, "<KMS_KEY_ID>", kmsKeyId, -1)
		}
		newContents = strings.Replace(newContents, "<CLUSTER_NAME>", clusterName, -1)

		if argocdOidcClientId != "" {
			newContents = strings.Replace(newContents, "<ARGOCD_OIDC_CLIENT_ID>", argocdOidcClientId, -1)
		}

		if viper.GetBool("create.terraformapplied.gitlab") {
			newContents = strings.Replace(newContents, "<AWS_HOSTED_ZONE_NAME>", hostedZoneName, -1)
			newContents = strings.Replace(newContents, "<AWS_DEFAULT_REGION>", region, -1)
			newContents = strings.Replace(newContents, "<AWS_ACCOUNT_ID>", awsAccountId, -1)
		}

		err = ioutil.WriteFile(path, []byte(newContents), 0)
		if err != nil {
			log.Panic(err)
		}

	}

	return nil
}

// SetupViper handles Viper config file. If config file doesn't exist, create, in case the file is available, use it.
func SetupViper(config *configs.Config) error {

	viperConfigFile := config.KubefirstConfigFilePath

	if _, err := os.Stat(viperConfigFile); errors.Is(err, os.ErrNotExist) {
		log.Printf("Config file not found, creating a blank one: %s \n", viperConfigFile)
		err = os.WriteFile(viperConfigFile, []byte("createdBy: installer\n\n"), 0700)
		if err != nil {
			return fmt.Errorf("unable to create blank config file, error is: %s", err)
		}
	}

	viper.SetConfigFile(viperConfigFile)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv() // read in environment variables that match

	// if a config file is found, read it in.
	err := viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("unable to read config file, error is: %s", err)
	}

	log.Println("Using config file:", viper.ConfigFileUsed())

	return nil
}

func CreateFile(fileName string, fileContent []byte) error {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error creating file: %s", err)
	}
	defer file.Close()
	_, err = file.Write(fileContent)
	if err != nil {
		return fmt.Errorf("unable to write the file: %s", err)
	}
	return nil
}
