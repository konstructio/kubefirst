package pkg

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
	yaml2 "gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RegistryAddon struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Annotations struct {
			AddonsKubefirstIoName string `yaml:"addons.kubefirst.io/name"`
		} `yaml:"annotations"`
	} `yaml:"metadata"`
}

// Detokenize - Translate tokens by values on a given path
func Detokenize(path string) {

	err := filepath.Walk(path, DetokenizeDirectory)
	if err != nil {
		log.Panic(err)
	}
}

// DetokenizeDirectory - Translate tokens by values on a directory level.
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

		var registryAddon RegistryAddon
		enableCheck := false
		removeFile := false

		err = yaml2.Unmarshal(read, &registryAddon)
		if err != nil {
			log.Println("trying read the file in yaml format: ", path, err)
		} else {
			enableCheck = true
		}

		//reading the addons list
		addons := viper.GetStringSlice("addons")
		log.Println("it is a yaml file, processing:", path)

		if enableCheck {
			if !slices.Contains(addons, registryAddon.Metadata.Annotations.AddonsKubefirstIoName) {
				log.Println("check if we need remove due unmatch annotation with k1 addons list: ", registryAddon.Metadata.Annotations)
				r := RegistryAddon{}
				if registryAddon.Metadata.Annotations != r.Metadata.Annotations {
					removeFile = true
					log.Println("yes, this file will be removed")
				} else {
					log.Println("no, this file will not be removed")
				}
			}
		}

		// todo should Detokenize be a switch statement based on a value found in viper?
		gitlabConfigured := viper.GetBool("gitlab.keyuploaded")
		githubConfigured := viper.GetBool("github.enabled")

		newContents := ""

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
		githubRepoOwner := viper.GetString(("github.owner"))
		githubRepoHost := viper.GetString(("github.host"))
		githubUser := viper.GetString(("github.user"))
		githubOrg := viper.GetString(("github.org"))

		//TODO:  We need to fix this
		githubToken := os.Getenv("GITHUB_AUTH_TOKEN")
		//TODO: Make this more clear
		isGithubMode := viper.GetBool("github.enabled")
		//todo: get from viper
		gitopsRepo := "gitops"

		newContents = strings.Replace(newContents, "<GITHUB_USER>", githubUser, -1)
		newContents = strings.Replace(newContents, "<GITHUB_TOKEN>", githubToken, -1)

		if gitlabConfigured {
			newContents = strings.Replace(string(read), "ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops", fmt.Sprintf("https://gitlab.%s/kubefirst/gitops.git", viper.GetString("aws.hostedzonename")), -1)
		} else if githubConfigured {
			newContents = strings.Replace(string(read), "https://gitlab.<AWS_HOSTED_ZONE_NAME>/kubefirst/gitops", "git@github.com:"+githubRepoOwner+"/"+"gitops", -1)
		} else {
			newContents = strings.Replace(string(read), "https://gitlab.<AWS_HOSTED_ZONE_NAME>/kubefirst/gitops.git", "ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops", -1)
		}

		var repoPathHTTPS string
		var repoPathSSH string
		var repoPathPrefered string

		if isGithubMode {
			repoPathHTTPS = "https://" + githubRepoHost + "/" + githubRepoOwner + "/" + gitopsRepo
			repoPathSSH = "git@" + githubRepoHost + "/" + githubRepoOwner + "/" + gitopsRepo
			repoPathPrefered = repoPathSSH
			newContents = strings.Replace(newContents, "<CHECKOUT_CWFT_TEMPLATE>", "git-checkout-with-gitops-ssh", -1)
			newContents = strings.Replace(newContents, "<GIT_REPO_RUNNER_NS>", "github-runner", -1)
			newContents = strings.Replace(newContents, "<GIT_REPO_RUNNER_NAME>", "github-runner", -1)
		} else {
			//not github = GITLAB
			repoPathHTTPS = "https://gitlab." + hostedZoneName + "/kubefirst/" + gitopsRepo
			repoPathSSH = "git@gitlab." + hostedZoneName + "/kubefirst/" + gitopsRepo
			//gitlab prefer HTTPS - for general use
			repoPathPrefered = repoPathHTTPS
			newContents = strings.Replace(newContents, "<CHECKOUT_CWFT_TEMPLATE>", "git-checkout-with-gitops", -1)
			newContents = strings.Replace(newContents, "<GIT_REPO_RUNNER_NS>", "default", -1)
			newContents = strings.Replace(newContents, "<GIT_REPO_RUNNER_NAME>", "gitlab-runner", -1)
		}
		repoPathNoProtocol := strings.Replace(repoPathHTTPS, "https://", "", -1)

		//for enforcing HTTPS
		newContents = strings.Replace(newContents, "<FULL_REPO_GITOPS_URL_HTTPS>", repoPathHTTPS, -1)
		newContents = strings.Replace(newContents, "<FULL_REPO_GITOPS_URL_NO_HTTPS>", repoPathNoProtocol, -1)
		//for enforcing SSH
		newContents = strings.Replace(newContents, "<FULL_REPO_GITOPS_URL_SSH>", repoPathSSH, -1)
		//gitlab prefer HTTPS - for general use
		newContents = strings.Replace(newContents, "<FULL_REPO_GITOPS_URL>", repoPathPrefered, -1)

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
		newContents = strings.Replace(newContents, "<ORG>", githubOrg, -1)
		newContents = strings.Replace(newContents, "<GITHUB_ORG>", githubOrg, -1)
		newContents = strings.Replace(newContents, "<GITHUB_HOST>", githubRepoHost, -1)
		newContents = strings.Replace(newContents, "<GITHUB_OWNER>", githubRepoOwner, -1)
		newContents = strings.Replace(newContents, "<GITHUB_USER>", githubUser, -1)
		newContents = strings.Replace(newContents, "<GITHUB_TOKEN>", githubToken, -1)

		newContents = strings.Replace(newContents, "<REPO_GITOPS>", "gitops", -1)

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

		if removeFile {
			err = os.Remove(path)
			if err != nil {
				log.Panic(err)
			}
		} else {
			err = ioutil.WriteFile(path, []byte(newContents), 0)
			if err != nil {
				log.Panic(err)
			}
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

// CreateFile - Create a file with its contents
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

// CreateFullPath - Create path and its parents
func CreateFullPath(p string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(p), 0777); err != nil {
		return nil, err
	}
	return os.Create(p)
}

// AwaitValidLetsEncryptCertificateNTimes do maxTimes attempts until it gets a valid Let's Encrypt certificate.
func AwaitValidLetsEncryptCertificateNTimes(domain string, dryRun bool, maxTimes int) (bool, error) {
	log.Println("AwaitValidTLSCertificateNTimes called")
	if dryRun {
		log.Printf("[#99] Dry-run mode, AwaitValidTLSCertificateNTimes skipped.")
		return true, nil
	}
	for i := 0; i < maxTimes; i++ {
		validCertificate, err := IsLetsEncryptCertificateDomain(domain)
		if err != nil {
			return false, err
		}

		if validCertificate {
			return true, nil
		}

		log.Printf(
			"domain (%q) is still not returning a valid Let's Encrypt certificate, attempt(%d of %d)",
			domain,
			i,
			maxTimes,
		)
		time.Sleep(time.Second * 10)

	}

	return false, fmt.Errorf("unable to have a valid Let's Encrypt certificate for the domain %q", domain)
}

// IsLetsEncryptCertificateDomain check if there is a Let's Encrypt certificate for the required domain. It's a simple
// validation and doesn't check in depth if the certificate is valid or invalid.
func IsLetsEncryptCertificateDomain(domain string) (bool, error) {

	sslPort := ":443"
	conn, err := tls.Dial("tcp", domain+sslPort, nil)
	if err != nil {
		return false, fmt.Errorf("unable to connect to the required host %q, error is: %v", domain, err)
	}

	err = conn.VerifyHostname(domain)
	if err != nil {
		return false, errors.New("unable to verify hostname")
	}

	if len(conn.ConnectionState().PeerCertificates) == 0 {
		return false, errors.New("there isn't a valid certificate to validate")
	}

	issuer := conn.ConnectionState().PeerCertificates[0].Issuer

	if len(issuer.Organization) == 0 {
		return false, errors.New("there isn't a valid organization in the certificate to be validated")
	}

	if issuer.Organization[0] != "Let's Encrypt" {
		return false, errors.New("certificate issuer isn't Let's Encrypt")
	}

	return true, nil
}
