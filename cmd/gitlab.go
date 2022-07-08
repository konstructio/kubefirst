package cmd

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"syscall"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

func gitlabGeneratePersonalAccessToken(gitlabPodName string) {
	kPortForward := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "gitlab", "port-forward", "svc/gitlab-webservice-default", "8888:8080")
	kPortForward.Stdout = os.Stdout
	kPortForward.Stderr = os.Stderr
	err := kPortForward.Start()
	defer kPortForward.Process.Signal(syscall.SIGTERM)
	if err != nil {
		log.Panicf("error: failed to port-forward to gitlab %s", err)
	}

	log.Println("generating gitlab personal access token on pod: ", gitlabPodName)

	id := uuid.New()
	gitlabToken := id.String()[:20]

	k := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "gitlab", "exec", gitlabPodName, "--", "gitlab-rails", "runner", fmt.Sprintf("token = User.find_by_username('root').personal_access_tokens.create(scopes: [:write_registry, :write_repository, :api], name: 'Automation token'); token.set_token('%s'); token.save!", gitlabToken))
	k.Stdout = os.Stdout
	k.Stderr = os.Stderr
	err = k.Run()
	if err != nil {
		log.Panicf("error running exec against %s to generate gitlab personal access token for root user", gitlabPodName)
	}

	viper.Set("gitlab.token", gitlabToken)
	viper.WriteConfig()

	log.Println("gitlab personal access token generated", gitlabToken)
}

func uploadGitlabSSHKey(gitlabToken string) {
	data := url.Values{
		"title": {"kubefirst"},
		"key":   {viper.GetString("botpublickey")},
	}

	gitlabUrlBase := viper.GetString("gitlab.local.service")

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := http.PostForm(gitlabUrlBase+"/api/v4/user/keys?private_token="+gitlabToken, data)
	if err != nil {
		log.Panicf("error: failed to upload ssh key to gitlab")
	}
	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	log.Println("ssh public key uploaded to gitlab")
	viper.Set("gitlab.keyuploaded", true)
	viper.WriteConfig()
}
