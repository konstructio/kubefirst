/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package pkg

import (
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/internal/progressPrinter"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/spf13/viper"
)

func CreateDirIfNotExist(dir string) error {
	if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		err = os.Mkdir(dir, 0777)
		if err != nil {
			return err
		}
	}
	return nil
}

func RemoveSubdomainV2(domainName string) (string, error) {

	log.Info().Msgf("original domain name: %s", domainName)

	domainName = strings.TrimRight(domainName, ".")
	domainSlice := strings.Split(domainName, ".")
	domainName = strings.Join([]string{domainSlice[len(domainSlice)-2], domainSlice[len(domainSlice)-1]}, ".")

	log.Info().Msgf("adjusted domain name:  %s", domainName)

	return domainName, nil
}

// SetupViper handles Viper config file. If config file doesn't exist, create, in case the file is available, use it.
func SetupViper(config *configs.Config) error {

	viperConfigFile := config.KubefirstConfigFilePath

	if _, err := os.Stat(viperConfigFile); errors.Is(err, os.ErrNotExist) {
		log.Printf("Config file not found, creating a blank one: %s \n", viperConfigFile)
		err = os.WriteFile(viperConfigFile, []byte(""), 0700)
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

	log.Info().Msgf("Using config file: %s", viper.ConfigFileUsed())

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

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func GenerateClusterID() string {
	return StringWithCharset(6, charset)
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
		return fmt.Errorf("rawURL cannot be empty string")
	}

	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil || parsedURL == nil {
		return fmt.Errorf("the URL (%s) is invalid, error = %v", rawURL, err)
	}
	return nil
}

// ValidateK1Folder receives a folder path, and expects the Kubefirst configuration folder doesn't contain "argocd-init-values.yaml" and/or "gitops/" folder.
// It follows this validation order:
//   - If folder doesn't exist, try to create it (happy path)
//   - If folder exists, and has "argocd-init-values.yaml" and/or "gitops/", abort and return error describing the issue and what should be done
func ValidateK1Folder(folderPath string) error {
	hasLeftOvers := false

	if _, err := os.Stat(folderPath); errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(folderPath, os.ModePerm); err != nil {
			return fmt.Errorf("info: could not create directory %q - error: %s", folderPath, err)
		}
		// folder was just created, no further validation required
		return nil
	}

	_, err := os.Stat(fmt.Sprintf("%s/argocd-init-values.yaml", folderPath))
	if err == nil {
		log.Debug().Msg("found argocd-init-values.yaml file")
		hasLeftOvers = true
	}

	_, err = os.Stat(fmt.Sprintf("%s/gitops", folderPath))
	if err == nil {
		log.Debug().Msg("found git-ops path")
		hasLeftOvers = true
	}

	if hasLeftOvers {
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

// ReplaceFileContent receives a file path, oldContent and newContent. oldContent is the previous value that is in the
// file, newContent is the new content you want to replace.
//
// Example:
//
//	err := ReplaceFileContent(vaultMainFile, "http://127.0.0.1:9000", "http://minio.minio.svc.cluster.local:9000")
func ReplaceFileContent(filPath string, oldContent string, newContent string) error {

	file, err := os.ReadFile(filPath)
	if err != nil {
		return err
	}

	updatedLine := strings.Replace(string(file), oldContent, newContent, -1)

	if err = os.WriteFile(filPath, []byte(updatedLine), 0); err != nil {
		return err
	}

	return nil
}

// UpdateTerraformS3BackendForK8sAddress during the installation process, Terraform must reach port-forwarded resources
// to be able to communicate with the services. When Kubefirst finish the installation, and Terraform needs to
// communicate with the services, it must use the internal Kubernetes addresses.
func UpdateTerraformS3BackendForK8sAddress(k1Dir string) error {

	// todo: create a function for file content replacement
	vaultMainFile := fmt.Sprintf("%s/gitops/terraform/vault/main.tf", k1Dir)
	if err := ReplaceFileContent(
		vaultMainFile,
		MinioURL,
		"http://minio.minio.svc.cluster.local:9000",
	); err != nil {
		return err
	}

	// update GitHub Terraform content
	if viper.GetString("git-provider") == "github" {
		fullPathKubefirstGitHubFile := fmt.Sprintf("%s/gitops/terraform/users/kubefirst-github.tf", k1Dir)
		if err := ReplaceFileContent(
			fullPathKubefirstGitHubFile,
			MinioURL,
			"http://minio.minio.svc.cluster.local:9000",
		); err != nil {
			return err
		}

		// change remote-backend.tf
		fullPathRemoteBackendFile := fmt.Sprintf("%s/gitops/terraform/github/remote-backend.tf", k1Dir)
		if err := ReplaceFileContent(
			fullPathRemoteBackendFile,
			MinioURL,
			"http://minio.minio.svc.cluster.local:9000",
		); err != nil {
			return err
		}
	}

	return nil
}

// UpdateTerraformS3BackendForLocalhostAddress during the destroy process, Terraform must reach port-forwarded resources
// to be able to communicate with the services.
func UpdateTerraformS3BackendForLocalhostAddress() error {

	config := configs.ReadConfig()

	// todo: create a function for file content replacement
	vaultMainFile := fmt.Sprintf("%s/gitops/terraform/vault/main.tf", config.K1FolderPath)
	if err := ReplaceFileContent(
		vaultMainFile,
		"http://minio.minio.svc.cluster.local:9000",
		MinioURL,
	); err != nil {
		return err
	}

	gitProvider := viper.GetString("git-provider")
	// update GitHub Terraform content
	if gitProvider == "github" {
		fullPathKubefirstGitHubFile := fmt.Sprintf("%s/gitops/terraform/users/kubefirst-github.tf", config.K1FolderPath)
		if err := ReplaceFileContent(
			fullPathKubefirstGitHubFile,
			"http://minio.minio.svc.cluster.local:9000",
			MinioURL,
		); err != nil {
			return err
		}

		// change remote-backend.tf
		fullPathRemoteBackendFile := fmt.Sprintf("%s/gitops/terraform/github/remote-backend.tf", config.K1FolderPath)
		if err := ReplaceFileContent(
			fullPathRemoteBackendFile,
			"http://minio.minio.svc.cluster.local:9000",
			MinioURL,
		); err != nil {
			log.Error().Err(err).Msg("")
		}
	}

	return nil
}

// todo: deprecate cmd.informUser
func InformUser(message string, silentMode bool) {
	// if in silent mode, send message to the screen
	// silent mode will silent most of the messages, this function is not frequently called
	if silentMode {
		_, err := fmt.Fprintln(os.Stdout, message)
		if err != nil {
			log.Error().Err(err).Msg("")
		}
		return
	}
	log.Info().Msg(message)
	progressPrinter.LogMessage(fmt.Sprintf("- %s", message))
}

// OpenBrowser opens the browser with the given URL
func OpenBrowser(url string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		if err = exec.Command("xdg-open", url).Start(); err != nil {
			return err
		}
	case "windows":
		if err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start(); err != nil {
			return err
		}
	case "darwin":
		if err = exec.Command("open", url).Start(); err != nil {
			return err
		}
	default:
		err = fmt.Errorf("unable to load the browser, unsupported platform")
		return err
	}

	return nil
}

// todo: this is temporary
func IsConsoleUIAvailable(url string) error {
	attempts := 10
	httpClient := http.DefaultClient
	for i := 0; i < attempts; i++ {

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			log.Printf("unable to reach %q (%d/%d)", url, i+1, attempts)
			time.Sleep(5 * time.Second)
			continue
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Printf("unable to reach %q (%d/%d)", url, i+1, attempts)
			time.Sleep(5 * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			log.Info().Msg("console UI is up and running")
			return nil
		}

		log.Info().Msg("waiting UI console to be ready")
		time.Sleep(5 * time.Second)
	}

	return nil
}

func OpenLogFile(path string) (*os.File, error) {
	logFile, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	return logFile, nil
}

// GetFileContent receives a file path, and return its content.
func GetFileContent(filePath string) ([]byte, error) {

	// check if file exists
	if _, err := os.Stat(filePath); err != nil && os.IsNotExist(err) {
		return nil, err
	}

	byteData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return byteData, nil
}

type CertificateAppList struct {
	Namespace string
	AppName   string
}

func GetCertificateAppList() []CertificateAppList {

	certificateAppList := []CertificateAppList{
		{
			Namespace: "argo",
			AppName:   "argo",
		},
		{
			Namespace: "argocd",
			AppName:   "argocd",
		},
		{
			Namespace: "atlantis",
			AppName:   "atlantis",
		},
		{
			Namespace: "chartmuseum",
			AppName:   "chartmuseum",
		},
		{
			Namespace: "vault",
			AppName:   "vault",
		},
		{
			Namespace: "minio",
			AppName:   "minio",
		},
		{
			Namespace: "minio",
			AppName:   "minio-console",
		},
		{
			Namespace: "kubefirst",
			AppName:   "kubefirst",
		},
		{
			Namespace: "development",
			AppName:   "metaphor-development",
		},
		{
			Namespace: "staging",
			AppName:   "metaphor-staging",
		},
		{
			Namespace: "production",
			AppName:   "metaphor-production",
		},
	}

	return certificateAppList
}

// FindStringInSlice takes []string and returns true if the supplied string is in the slice.
func FindStringInSlice(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func ResetK1Dir(k1Dir string) error {

	if _, err := os.Stat(k1Dir + "/argo-workflows"); !os.IsNotExist(err) {
		// path/to/whatever exists
		err := os.RemoveAll(k1Dir + "/argo-workflows")
		if err != nil {
			return fmt.Errorf("unable to delete %q folder, error: %s", k1Dir+"/argo-workflows", err)
		}
	}

	if _, err := os.Stat(k1Dir + "/gitops"); !os.IsNotExist(err) {
		err := os.RemoveAll(k1Dir + "/gitops")
		if err != nil {
			return fmt.Errorf("unable to delete %q folder, error: %s", k1Dir+"/gitops", err)
		}
	}
	if _, err := os.Stat(k1Dir + "/metaphor"); !os.IsNotExist(err) {
		err := os.RemoveAll(k1Dir + "/metaphor")
		if err != nil {
			return fmt.Errorf("unable to delete %q folder, error: %s", k1Dir+"/metaphor", err)
		}
	}
	// todo look at logic to not re-download
	if _, err := os.Stat(k1Dir + "/tools"); !os.IsNotExist(err) {
		err = os.RemoveAll(k1Dir + "/tools")
		if err != nil {
			return fmt.Errorf("unable to delete %q folder, error: %s", k1Dir+"/tools", err)
		}
	}
	//* files
	//! this might fail with an adjustment made to validate
	if _, err := os.Stat(k1Dir + "/argocd-init-values.yaml"); !os.IsNotExist(err) {
		err = os.Remove(k1Dir + "/argocd-init-values.yaml")
		if err != nil {
			return fmt.Errorf("unable to delete %q folder, error: %s", k1Dir+"/argocd-init-values.yaml", err)
		}
	}

	return nil

}
