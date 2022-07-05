package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"github.com/spf13/viper"

	"os/exec"
	"time"

	"net/url"
	"net/http"
	"encoding/json"

	"github.com/google/uuid"
	"bytes"
	"encoding/base64"
	"github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	b64 "encoding/base64"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"  
	"html/template"
	"github.com/ghodss/yaml"
	"context"
)

func helmInstallArgocd(home string, kubeconfigPath string) {
	if !viper.GetBool("create.argocd.helm") {
		if dryrunMode {
			log.Printf("[#99] Dry-run mode, helmInstallArgocd skipped.")
			return
		}
		// ! commenting out until a clean execution is necessary // create namespace
		helmRepoAddArgocd := exec.Command(helmClientPath, "--kubeconfig", kubeconfigPath, "repo", "add", "argo", "https://argoproj.github.io/argo-helm")
		helmRepoAddArgocd.Stdout = os.Stdout
		helmRepoAddArgocd.Stderr = os.Stderr
		err := helmRepoAddArgocd.Run()
		if err != nil {
			log.Panicf("error: could not run helm repo add %s", err)
		}

		helmRepoUpdate := exec.Command(helmClientPath, "--kubeconfig", kubeconfigPath, "repo", "update")
		helmRepoUpdate.Stdout = os.Stdout
		helmRepoUpdate.Stderr = os.Stderr
		err = helmRepoUpdate.Run()
		if err != nil {
			log.Panicf("error: could not helm repo update %s", err)
		}

		helmInstallArgocdCmd := exec.Command(helmClientPath, "--kubeconfig", kubeconfigPath, "upgrade", "--install", "argocd", "--namespace", "argocd", "--create-namespace", "--wait", "--values", fmt.Sprintf("%s/.kubefirst/argocd-init-values.yaml", home), "argo/argo-cd")
		helmInstallArgocdCmd.Stdout = os.Stdout
		helmInstallArgocdCmd.Stderr = os.Stderr
		err = helmInstallArgocdCmd.Run()
		if err != nil {
			log.Panicf("error: could not helm install argocd command %s", err)
		}

		viper.Set("create.argocd.helm", true)
		err = viper.WriteConfig()
		if err != nil {
			log.Panicf("error: could not write to viper config")
		}
	}
}

func awaitGitlab() {
	log.Println("awaitGitlab called")
	if dryrunMode {
		log.Printf("[#99] Dry-run mode, awaitGitlab skipped.")
		return
	}	
	max := 200
	for i := 0; i < max; i++ {
		hostedZoneName := viper.GetString("aws.hostedzonename")
		resp, _ := http.Get(fmt.Sprintf("https://gitlab.%s", hostedZoneName))
		if resp != nil && resp.StatusCode == 200 {
			log.Println("gitlab host resolved, 30 second grace period required...")
			time.Sleep(time.Second * 30)
			i = max
		} else {
			log.Println("gitlab host not resolved, sleeping 10s")
			time.Sleep(time.Second * 10)
		}
	}
}

func produceGitlabTokens(){
	//TODO: Should this step be skipped if already executed?
	log.Println("discovering gitlab toolbox pod")	
	if dryrunMode {
		log.Printf("[#99] Dry-run mode, produceGitlabTokens skipped.")
		return
	}
	var outb, errb bytes.Buffer
	k := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "gitlab", "get", "pod", "-lapp=toolbox", "-o", "jsonpath='{.items[0].metadata.name}'")
	k.Stdout = &outb
	k.Stderr = &errb
	err := k.Run()
	if err != nil {
		log.Println("failed to call k.Run() to get gitlab pod: ", err)
	}
	gitlabPodName := outb.String()
	gitlabPodName = strings.Replace(gitlabPodName, "'", "", -1)
	log.Println("gitlab pod", gitlabPodName)

	gitlabToken := viper.GetString("gitlab.token")
	if gitlabToken == "" {

		log.Println("getting gitlab personal access token")

		id := uuid.New()
		gitlabToken = id.String()[:20]

		k = exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "gitlab", "exec", gitlabPodName, "--", "gitlab-rails", "runner", fmt.Sprintf("token = User.find_by_username('root').personal_access_tokens.create(scopes: [:write_registry, :write_repository, :api], name: 'Automation token'); token.set_token('%s'); token.save!", gitlabToken))
		k.Stdout = os.Stdout
		k.Stderr = os.Stderr
		err = k.Run()
		if err != nil {
			log.Println("failed to call k.Run() to set gitlab token: ", err)
		}

		viper.Set("gitlab.token", gitlabToken)
		viper.WriteConfig()

		log.Println("gitlabToken", gitlabToken)
	}

	gitlabRunnerToken := viper.GetString("gitlab.runnertoken")
	if gitlabRunnerToken == "" {

		log.Println("getting gitlab runner token")

		var tokenOut, tokenErr bytes.Buffer
		k = exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "gitlab", "get", "secret", "gitlab-gitlab-runner-secret", "-o", "jsonpath='{.data.runner-registration-token}'")
		k.Stdout = &tokenOut
		k.Stderr = &tokenErr
		err = k.Run()
		if err != nil {
			log.Println("failed to call k.Run() to get gitlabRunnerRegistrationToken: ", err)
		}
		encodedToken := tokenOut.String()
		log.Println(encodedToken)
		encodedToken = strings.Replace(encodedToken, "'", "", -1)
		log.Println(encodedToken)
		gitlabRunnerRegistrationTokenBytes, err := base64.StdEncoding.DecodeString(encodedToken)
		gitlabRunnerRegistrationToken := string(gitlabRunnerRegistrationTokenBytes)
		log.Println(gitlabRunnerRegistrationToken)
		if err != nil {
			panic(err)
		}
		viper.Set("gitlab.runnertoken", gitlabRunnerRegistrationToken)
		viper.WriteConfig()
		log.Println("gitlabRunnerRegistrationToken", gitlabRunnerRegistrationToken)
	}

}

func applyGitlabTerraform(directory string){
	if !viper.GetBool("create.terraformapplied.gitlab") {
		log.Println("Executing applyGitlabTerraform")
		if dryrunMode {
			log.Printf("[#99] Dry-run mode, applyGitlabTerraform skipped.")
			return
		}		
		// Prepare for terraform gitlab execution
		os.Setenv("GITLAB_TOKEN", viper.GetString("gitlab.token"))
		os.Setenv("GITLAB_BASE_URL", fmt.Sprintf("https://gitlab.%s", viper.GetString("aws.domainname")))

		directory = fmt.Sprintf("%s/.kubefirst/gitops/terraform/gitlab", home)
		err := os.Chdir(directory)
		if err != nil {
			log.Println("error changing dir")
		}
		execShellReturnStrings(terraformPath, "init")
		execShellReturnStrings(terraformPath, "apply", "-auto-approve")
		viper.Set("create.terraformapplied.gitlab", true)
		viper.WriteConfig()
	} else {
		log.Println("Skipping: applyGitlabTerraform")
	}
}



func gitlabKeyUpload(){
	// upload ssh public key	
	if !viper.GetBool("gitlab.keyuploaded") {
		log.Println("Executing gitlabKeyUpload")
		if dryrunMode {
			log.Printf("[#99] Dry-run mode, gitlabKeyUpload skipped.")
			return
		}		
		log.Println("uploading ssh public key to gitlab")
		gitlabToken := viper.GetString("gitlab.token")
		data := url.Values{
			"title": {"kubefirst"},
			"key":   {viper.GetString("botpublickey")},
		}

		gitlabUrlBase := fmt.Sprintf("https://gitlab.%s", viper.GetString("aws.domainname"))

		resp, err := http.PostForm(gitlabUrlBase+"/api/v4/user/keys?private_token="+gitlabToken, data)
		if err != nil {
			log.Fatal(err)
		}
		var res map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&res)
		log.Println(res)
		log.Println("ssh public key uploaded to gitlab")
		viper.Set("gitlab.keyuploaded", true)
		viper.WriteConfig()
	} else {
		log.Println("Skipping: gitlabKeyUpload")
		log.Println("ssh public key already uploaded to gitlab")
	}
}

func pushGitopsToGitLab() {
	if dryrunMode {
		log.Printf("[#99] Dry-run mode, pushGitopsToGitLab skipped.")
		return
	}
	  
	//TODO: should this step to be skipped if already executed?
	domain := viper.GetString("aws.hostedzonename")

	detokenize(fmt.Sprintf("%s/.kubefirst/gitops", home))
	directory := fmt.Sprintf("%s/.kubefirst/gitops", home)

	repo, err := git.PlainOpen(directory)
	if err != nil {
		log.Panicf("error opening the directory ", directory, err)
	}

	//upstream := fmt.Sprintf("ssh://gitlab.%s:22:kubefirst/gitops", viper.GetString("aws.hostedzonename"))
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
			Email: "kubefirst-bot@kubefirst.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		log.Panicf("error committing changes", err)
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
		log.Panicf("error pushing to remote", err)
	}

}

func changeRegistryToGitLab() {
	if !viper.GetBool("gitlab.registry") {
		if dryrunMode {
			log.Printf("[#99] Dry-run mode, changeRegistryToGitLab skipped.")
			return
		}

		type ArgocdGitCreds struct {
			PersonalAccessToken string
			URL                 string
			FullURL             string
		}

		pat := b64.StdEncoding.EncodeToString([]byte(viper.GetString("gitlab.token")))
		url := b64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("https://gitlab.%s/kubefirst/", viper.GetString("aws.hostedzonename"))))
		fullurl := b64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("https://gitlab.%s/kubefirst/gitops.git", viper.GetString("aws.hostedzonename"))))

		creds := ArgocdGitCreds{PersonalAccessToken: pat, URL: url, FullURL: fullurl}

		var argocdRepositoryAccessTokenSecret *v1.Secret
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			log.Panicf("error getting client from kubeconfig")
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Panicf("error getting kubeconfig for clientset")
		}
		argocdSecretClient = clientset.CoreV1().Secrets("argocd")

		var secrets bytes.Buffer

		c, err := template.New("creds-gitlab").Parse(`
      apiVersion: v1
      data:
        password: {{ .PersonalAccessToken }}
        url: {{ .URL }}
        username: cm9vdA==
      kind: Secret
      metadata:
        annotations:
          managed-by: argocd.argoproj.io
        labels:
          argocd.argoproj.io/secret-type: repo-creds
        name: creds-gitlab
        namespace: argocd
      type: Opaque
    `)
		if err := c.Execute(&secrets, creds); err != nil {
			log.Panicf("error executing golang template for git repository credentials template %s", err)
		}

		ba := []byte(secrets.String())
		err = yaml.Unmarshal(ba, &argocdRepositoryAccessTokenSecret)

		_, err = argocdSecretClient.Create(context.TODO(), argocdRepositoryAccessTokenSecret, metaV1.CreateOptions{})
		if err != nil {
			log.Panicf("error creating argocd repository credentials template secret %s", err)
		}

		var repoSecrets bytes.Buffer

		c, err = template.New("repo-gitlab").Parse(`
      apiVersion: v1
      data:
        project: ZGVmYXVsdA==
        type: Z2l0
        url: {{ .FullURL }}
      kind: Secret
      metadata:
        annotations:
          managed-by: argocd.argoproj.io
        labels:
          argocd.argoproj.io/secret-type: repository
        name: repo-gitlab
        namespace: argocd
      type: Opaque
    `)
		if err := c.Execute(&repoSecrets, creds); err != nil {
			log.Panicf("error executing golang template for gitops repository template %s", err)
		}

		ba = []byte(repoSecrets.String())
		err = yaml.Unmarshal(ba, &argocdRepositoryAccessTokenSecret)

		_, err = argocdSecretClient.Create(context.TODO(), argocdRepositoryAccessTokenSecret, metaV1.CreateOptions{})
		if err != nil {
			log.Panicf("error creating argocd repository connection secret %s", err)
		}

		k := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "argocd", "apply", "-f", fmt.Sprintf("%s/.kubefirst/gitops/components/gitlab/argocd-adopts-gitlab.yaml", home))
		k.Stdout = os.Stdout
		k.Stderr = os.Stderr
		err = k.Run()
		if err != nil {
			log.Panicf("failed to call execute kubectl apply of argocd patch to adopt gitlab: %s", err)
		}

		viper.Set("gitlab.registry", true)
		viper.WriteConfig()
	} else {
		log.Println("Skipping: changeRegistryToGitLab")
	}
}

func getArgocdAuthToken() string {
	kPortForward := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "argocd", "port-forward", "svc/argocd-server", "8080:8080")
	kPortForward.Stdout = os.Stdout
	kPortForward.Stderr = os.Stderr
	err := kPortForward.Start()
	defer kPortForward.Process.Signal(syscall.SIGTERM)
	if err != nil {
		log.Panicf("error: failed to port-forward to argocd %s", err)
	}

	url := "https://localhost:8080/api/v1/session"

	payload := strings.NewReader(fmt.Sprintf("{\n\t\"username\":\"admin\",\"password\":\"%s\"\n}", viper.GetString("argocd.admin.password")))

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		log.Fatal("error getting auth token from argocd ", err)
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// N.B.: when used in production, also check for redirect loops
			return nil
		},
	}

	res, err := client.Do(req)
	if err != nil {
		log.Fatal("error requesting auth token from argocd")
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal("error sending POST request to get argocd auth token :", err)
	}

	var dat map[string]interface{}

	if err := json.Unmarshal(body, &dat); err != nil {
		log.Panicf("error unmarshalling  %s", err)
	}
	token := dat["token"]
	viper.Set("argocd.admin.apitoken", token)
	viper.WriteConfig()

	return token.(string)

}

func syncArgocdApplication(applicationName, argocdAuthToken string) {
	if dryrunMode {
		log.Printf("[#99] Dry-run mode, syncArgocdApplication skipped.")
		return
	}
	kPortForward := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "argocd", "port-forward", "svc/argocd-server", "8080:8080")
	kPortForward.Stdout = os.Stdout
	kPortForward.Stderr = os.Stderr
	err := kPortForward.Start()
	defer kPortForward.Process.Signal(syscall.SIGTERM)
	if err != nil {
		log.Panicf("error: failed to port-forward to argocd %s", err)
	}

	// todo need to replace this with a curl wrapper and see if it WORKS

	url := fmt.Sprintf("https://localhost:8080/api/v1/applications/%s/sync", applicationName)

	argoCdAppSync := exec.Command("curl", "-k", "-L", "-X", "POST", url, "-H", fmt.Sprintf("Authorization: Bearer %s", argocdAuthToken))
	argoCdAppSync.Stdout = os.Stdout
	argoCdAppSync.Stderr = os.Stderr
	err = argoCdAppSync.Run()
	if err != nil {
		log.Panicf("error: curl appSync failed failed %s", err)
	}
}