package pkg

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
	yaml2 "gopkg.in/yaml.v2"
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

	if viper.GetBool("github.enabled") && strings.Contains(path, "-gitlab.tf") {
		log.Println("github is enabled, removing gitlab terraform file:", path)
		err = os.Remove(path)
		if err != nil {
			log.Panic(err)
		}
		return nil
	}
	if !viper.GetBool("github.enabled") && strings.Contains(path, "-github.tf") {
		log.Println("gitlab is enabled, removing github terraform file:", path)
		err = os.Remove(path)
		if err != nil {
			log.Panic(err)
		}
		return nil
	}

	matched, err := filepath.Match("*", fi.Name())

	if err != nil {
		log.Panic(err)
	}

	if matched {
		read, err := os.ReadFile(path)
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

		//Please, don't remove comments on this file unless you added it
		// todo should Detokenize be a switch statement based on a value found in viper?
		gitlabConfigured := viper.GetBool("gitlab.keyuploaded")
		//githubConfigured := viper.GetBool("github.enabled")

		newContents := string(read)

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
		argocdOidcClientId := viper.GetString(("vault.oidc.argocd.client_id"))
		githubRepoHost := viper.GetString(("github.host"))
		githubRepoOwner := viper.GetString(("github.owner"))
		githubOrg := viper.GetString(("github.org"))
		githubUser := viper.GetString(("github.user"))

		//TODO:  We need to fix this
		githubToken := os.Getenv("GITHUB_AUTH_TOKEN")
		//TODO: Make this more clear
		isGithubMode := viper.GetBool("github.enabled")
		//todo: get from viper
		gitopsRepo := "gitops"
		repoPathHTTPSGitlab := "https://gitlab." + hostedZoneName + "/kubefirst/" + gitopsRepo

		newContents = strings.Replace(newContents, "<GITHUB_USER>", githubUser, -1)
		newContents = strings.Replace(newContents, "<GITHUB_TOKEN>", githubToken, -1)

		var repoPathHTTPS string
		var repoPathSSH string
		var repoPathPrefered string

		if isGithubMode {
			repoPathHTTPS = "https://" + githubRepoHost + "/" + githubRepoOwner + "/" + gitopsRepo
			repoPathSSH = "git@" + githubRepoHost + "/" + githubRepoOwner + "/" + gitopsRepo
			repoPathPrefered = repoPathSSH
			newContents = strings.Replace(newContents, "<CHECKOUT_CWFT_TEMPLATE>", "git-checkout-with-gitops-ssh", -1)
			newContents = strings.Replace(newContents, "<COMMIT_CWFT_TEMPLATE>", "git-commit-ssh", -1)
			newContents = strings.Replace(newContents, "<GIT_REPO_RUNNER_NS>", "github-runner", -1)
			newContents = strings.Replace(newContents, "<GIT_REPO_RUNNER_NAME>", "github-runner", -1)
		} else {
			//not github = GITLAB
			repoPathHTTPS = repoPathHTTPSGitlab
			repoPathSSH = "git@gitlab." + hostedZoneName + "/kubefirst/" + gitopsRepo
			//gitlab prefer HTTPS - for general use
			repoPathPrefered = repoPathHTTPS
			if gitlabConfigured {
				repoPathPrefered = repoPathHTTPSGitlab
				newContents = strings.Replace(newContents, "ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops", fmt.Sprintf("https://gitlab.%s/kubefirst/gitops.git", viper.GetString("aws.hostedzonename")), -1)
			} else {
				//Default start-soft-serve
				repoPathPrefered = "ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops"
			}
			newContents = strings.Replace(newContents, "<CHECKOUT_CWFT_TEMPLATE>", "git-checkout-with-gitops", -1)
			newContents = strings.Replace(newContents, "<COMMIT_CWFT_TEMPLATE>", "git-commit", -1)
			newContents = strings.Replace(newContents, "<GIT_REPO_RUNNER_NS>", "default", -1)
			newContents = strings.Replace(newContents, "<GIT_REPO_RUNNER_NAME>", "gitlab-runner", -1)
		}
		/*
			if gitlabConfigured {
				newContents = strings.Replace(string(read), "ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops", fmt.Sprintf("https://gitlab.%s/kubefirst/gitops.git", viper.GetString("aws.hostedzonename")), -1)
			} else if githubConfigured {
				newContents = strings.Replace(string(read), "https://gitlab.<AWS_HOSTED_ZONE_NAME>/kubefirst/gitops", "git@github.com:"+githubRepoOwner+"/"+"gitops", -1)
			} else {
				newContents = strings.Replace(string(read), repoPathHTTPSGitlab, "ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops", -1)
			}
		*/

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
			err = os.WriteFile(path, []byte(newContents), 0)
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

func randSeq(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func Random(seq int) string {
	rand.Seed(time.Now().UnixNano())
	return randSeq(seq)
}

// RemoveSubDomain receives a host and remove its subdomain, if exists.
func RemoveSubDomain(fullURL string) (string, error) {

	// add http if fullURL doesn't have it, this is for validation only, won't be used on http requests
	if !strings.HasPrefix(fullURL, "http") {
		fullURL = "https://" + fullURL
	}

	// check if received fullURL is valid before parsing it
	err := IsValidURL(fullURL)
	if err != nil {
		return "", err
	}

	// build URL
	fullPathURL, err := url.ParseRequestURI(fullURL)
	if err != nil {
		return "", err
	}

	splitHost := strings.Split(fullPathURL.Host, ".")

	if len(splitHost) < 2 {
		return "", fmt.Errorf("the fullURL (%s) is invalid", fullURL)
	}

	lastURLPart := splitHost[len(splitHost)-2:]
	hostWithSpace := strings.Join(lastURLPart, " ")
	// set fullURL only without subdomain
	fullPathURL.Host = strings.ReplaceAll(hostWithSpace, " ", ".")

	// build URL without subdomain
	result := fullPathURL.Scheme + "://" + fullPathURL.Host

	// check if new URL is still valid
	err = IsValidURL(result)
	if err != nil {
		return "", err
	}

	return fullPathURL.Host, nil
}

// IsValidURL checks if a URL is valid
func IsValidURL(rawURL string) error {

	if len(rawURL) == 0 {
		return errors.New("rawURL cannot be empty string")
	}

	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil || parsedURL == nil {
		return fmt.Errorf("the URL (%s) is invalid, error = %v", rawURL, err)
	}
	return nil
}

// ValidateK1Folder receives a folder path, and expect the Kubefirst configuration folder is empty. It follows this
// validation list:
//   - If folder doesn't exist, try to create it
//   - If folder exists, check if there are files
//   - If folder exists, and has files, inform the user that clean command should be called before a new init
func ValidateK1Folder(folderPath string) error {

	if _, err := os.Stat(folderPath); errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(folderPath, os.ModePerm); err != nil {
			return fmt.Errorf("info: could not create directory %q - error: %s", folderPath, err)
		}
		// folder was just created, no further validation required
		return nil
	}

	files, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}

	if len(files) != 0 {
		return fmt.Errorf("folder: %s has files that can be left overs from a previous installation, "+
			"please use kubefirst clean command to be ready for a new installation", folderPath)
	}

	return nil
}

// AwaitHostNTimes - Wait for a Host to return a 200
// - To return 200
// - To return true if host is ready, or false if not
// - Supports a number of times to test an endpoint
// - Supports the grace period after status 200 to wait before returning
func AwaitHostNTimes(url string, times int, gracePeriod time.Duration) {
	log.Printf("AwaitHostNTimes %d called with grace period of: %d seconds", times, gracePeriod)
	max := times
	for i := 0; i < max; i++ {
		resp, _ := http.Get(url)
		if resp != nil && resp.StatusCode == 200 {
			log.Printf("%s resolved, %s second grace period required...", url, gracePeriod)
			time.Sleep(time.Second * gracePeriod)
			return
		} else {
			log.Printf("%s not resolved, sleeping 10s", url)
			time.Sleep(time.Second * 10)
		}
	}
}

// type NgrokOutput struct {
// 	Tunnels []struct {
// 		PublicURL string `json:"public_url"`
// 	} `json:"tunnels"`
// 	URI string `json:"uri"`
// }

// func OpenNgrokTunnel() string {

// 	config := configs.ReadConfig()

// 	var ngrokOutb, ngrokErrb bytes.Buffer
// 	openNgrokTunnel := exec.Command(config.NgrokClientPath, "http", "4141")
// 	openNgrokTunnel.Stdout = &ngrokOutb
// 	openNgrokTunnel.Stderr = &ngrokErrb
// 	err := openNgrokTunnel.Start()
// 	url := "http://localhost:4040/api/tunnels"
// 	outb, _, err := ExecShellReturnStrings("curl", url)
// 	if err != nil {
// 		log.Panicf("error starting ngrok on port 4141: %s", err)
// 	}
// 	ngrokOutput := &NgrokOutput{}
// 	err = json.Unmarshal([]byte(outb), ngrokOutput)
// 	if err != nil {
// 		log.Println("error unmarshalling json from curl command ")
// 	}
// 	fmt.Println(ngrokOutput.Tunnels[0].PublicURL)
// 	return ngrokOutput.Tunnels[0].PublicURL
// }

// this is temporary code
func ReplaceS3Backend() error {

	config := configs.ReadConfig()

	vaultMainFile := fmt.Sprintf("%s/gitops/terraform/vault/main.tf", config.K1FolderPath)

	file, err := os.ReadFile(vaultMainFile)
	if err != nil {
		return err
	}
	newContents := strings.Replace(string(file), "http://127.0.0.1:9000", "http://minio.minio.svc.cluster.local:9000", -1)

	err = os.WriteFile(vaultMainFile, []byte(newContents), 0)
	if err != nil {
		return err
	}

	kubefirstGitHubFile := fmt.Sprintf("%s/gitops/terraform/users/kubefirst-github.tf", config.K1FolderPath)
	file2, err := os.ReadFile(kubefirstGitHubFile)
	if err != nil {
		return err
	}
	newContents2 := strings.Replace(string(file2), "http://127.0.0.1:9000", "http://minio.minio.svc.cluster.local:9000", -1)

	err = os.WriteFile(kubefirstGitHubFile, []byte(newContents2), 0)
	if err != nil {
		return err
	}

	return nil
}
