package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"github.com/spf13/viper"
	"syscall"
	"os/exec"
	"time"
	"crypto/tls"
	"net/url"
	"net/http"
	"encoding/json"
	"io/ioutil"
	"bytes"
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
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	log.Println("discovering gitlab toolbox pod")	
	if dryrunMode {
		log.Printf("[#99] Dry-run mode, produceGitlabTokens skipped.")
		return
	}
	time.Sleep(30 * time.Second)
	argocdSecretClient = clientset.CoreV1().Secrets("argocd")

	argocdPassword := getSecretValue(argocdSecretClient, "argocd-initial-admin-secret", "password")

	viper.Set("argocd.admin.password", argocdPassword)
	viper.WriteConfig()

	log.Println("discovering gitlab toolbox pod")

	gitlabPodsClient = clientset.CoreV1().Pods("gitlab")
	gitlabPodName := getPodNameByLabel(gitlabPodsClient, "toolbox")

	gitlabSecretClient = clientset.CoreV1().Secrets("gitlab")
	secrets, err := gitlabSecretClient.List(context.TODO(), metaV1.ListOptions{})

	var gitlabRootPasswordSecretName string

	for _, secret := range secrets.Items {
		if strings.Contains(secret.Name, "initial-root-password") {
			gitlabRootPasswordSecretName = secret.Name
			log.Println("gitlab initial root password secret name: ", gitlabRootPasswordSecretName)
		}
	}
	gitlabRootPassword := getSecretValue(gitlabSecretClient, gitlabRootPasswordSecretName, "password")

	viper.Set("gitlab.podname", gitlabPodName)
	viper.Set("gitlab.root.password", gitlabRootPassword)
	viper.WriteConfig()

	gitlabToken := viper.GetString("gitlab.token")

	if gitlabToken == "" {

		log.Println("generating gitlab personal access token")
		gitlabGeneratePersonalAccessToken(gitlabPodName)

	}

	gitlabRunnerToken := viper.GetString("gitlab.runnertoken")

	if gitlabRunnerToken == "" {

		log.Println("getting gitlab runner token")
		gitlabRunnerRegistrationToken := getSecretValue(gitlabSecretClient, "gitlab-gitlab-runner-secret", "runner-registration-token")
		viper.Set("gitlab.runnertoken", gitlabRunnerRegistrationToken)
		viper.WriteConfig()
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
		os.Setenv("GITLAB_BASE_URL", "http://localhost:8888")


		directory = fmt.Sprintf("%s/.kubefirst/gitops/terraform/gitlab", home)
		err := os.Chdir(directory)
		if err != nil {
			log.Panic("error: could not change directory to " + directory)
		}
		_,_,errInit := execShellReturnStrings(terraformPath, "init")
		if errInit != nil {
			panic(fmt.Sprintf("error: terraform init for gitlab failed %s", err))
		}
		_,_,errApply := execShellReturnStrings(terraformPath, "apply", "-auto-approve")
		if errApply != nil {
			panic(fmt.Sprintf("error: terraform apply for gitlab failed %s", err))
		}
		os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))
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
		log.Println("uploading ssh public key for gitlab user")
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
	if dryrunMode {
		log.Printf("[#99] Dry-run mode, getArgocdAuthToken skipped.")
		return "nothing"
	  }
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

func  destroyGitlabTerraform(){
	log.Println("\n\nTODO -- need to setup and argocd delete against registry and wait?\n\n")
	// kubeconfig := os.Getenv("HOME") + "/.kube/config"
	// config, err := argocdclientset.BuildConfigFromFlags("", kubeconfig)
	// argocdclientset, err := argocdclientset.NewForConfig(config)
	// if err != nil {
	// 	return nil, err
	// }

	//* should we git clone the gitops repo when destroy is run back to their
	//* local host to get the latest values of gitops

	os.Setenv("AWS_REGION", viper.GetString("aws.region"))
	os.Setenv("AWS_ACCOUNT_ID", viper.GetString("aws.accountid"))
	os.Setenv("HOSTED_ZONE_NAME", viper.GetString("aws.hostedzonename"))
	os.Setenv("GITLAB_TOKEN", viper.GetString("gitlab.token"))

	os.Setenv("TF_VAR_aws_account_id", viper.GetString("aws.accountid"))
	os.Setenv("TF_VAR_aws_region", viper.GetString("aws.region"))
	os.Setenv("TF_VAR_hosted_zone_name", viper.GetString("aws.hostedzonename"))

	directory := fmt.Sprintf("%s/.kubefirst/gitops/terraform/gitlab", home)	
	err := os.Chdir(directory)
	if err != nil {
		log.Panicf("error: could not change directory to " + directory)
	}

	os.Setenv("GITLAB_BASE_URL", "http://localhost:8888")

	if !skipGitlabTerraform {
		tfInitGitlabCmd := exec.Command(terraformPath, "init")
		tfInitGitlabCmd.Stdout = os.Stdout
		tfInitGitlabCmd.Stderr = os.Stderr
		err = tfInitGitlabCmd.Run()
		if err != nil {
			log.Panicf("failed to terraform init gitlab %s", err)
		}

		tfDestroyGitlabCmd := exec.Command(terraformPath, "destroy", "-auto-approve")
		tfDestroyGitlabCmd.Stdout = os.Stdout
		tfDestroyGitlabCmd.Stderr = os.Stderr
		err = tfDestroyGitlabCmd.Run()
		if err != nil {
			log.Panicf("failed to terraform destroy gitlab %s", err)
		}

		viper.Set("destroy.terraformdestroy.gitlab", true)
		viper.WriteConfig()
	}
}