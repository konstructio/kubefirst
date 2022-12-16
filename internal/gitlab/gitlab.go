package gitlab

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kubefirst/kubefirst/internal/gitClient"
	internalSSH "github.com/kubefirst/kubefirst/internal/ssh"

	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/aws"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/uuid"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

// GenerateKey -  generate public and private keys to be consumed by GitLab.
func GenerateKey() (string, string, error) {
	reader := rand.Reader
	bitSize := 2048

	key, err := rsa.GenerateKey(reader, bitSize)
	if err != nil {
		return "", "", err
	}

	pub, err := ssh.NewPublicKey(key.Public())
	if err != nil {
		return "", "", err
	}
	publicKey := string(ssh.MarshalAuthorizedKey(pub))
	// encode RSA key
	privateKey := string(pem.EncodeToMemory(
		&pem.Block{
			Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	))

	return publicKey, privateKey, nil
}

// GitlabGeneratePersonalAccessToken - Generate a Access Token for Gitlab
func GitlabGeneratePersonalAccessToken(gitlabPodName string) {
	config := configs.ReadConfig()

	log.Info().Msgf("generating gitlab personal access token on pod: %s", gitlabPodName)

	id := uuid.New()
	gitlabToken := id.String()[:20]

	_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "gitlab", "exec", gitlabPodName, "--", "gitlab-rails", "runner", fmt.Sprintf("token = User.find_by_username('root').personal_access_tokens.create(scopes: [:write_registry, :write_repository, :api], name: 'Automation token'); token.set_token('%s'); token.save!", gitlabToken))
	if err != nil {
		log.Panic().Msgf("error running exec against %s to generate gitlab personal access token for root user %s", gitlabPodName)
	}

	viper.Set("gitlab.token", gitlabToken)
	viper.WriteConfig()

	log.Info().Msgf("gitlab personal access token generated", gitlabToken)
}

// PushGitOpsToGitLab - Push GitOps to Gitlab repository
// Use repo loaded from `init“
func PushGitOpsToGitLab(dryRun bool) {
	cfg := configs.ReadConfig()
	if dryRun {
		log.Printf("[#99] Dry-run mode, PushGitOpsToGitLab skipped.")
		return
	}

	//TODO: should this step to be skipped if already executed?
	domain := viper.GetString("aws.hostedzonename")

	pkg.Detokenize(fmt.Sprintf("%s/gitops", cfg.K1FolderPath))
	directory := fmt.Sprintf("%s/gitops", cfg.K1FolderPath)

	repo, err := git.PlainOpen(directory)
	if err != nil {
		log.Panic().Msgf("error opening the directory %s:  %s", directory, err)
	}

	upstream := fmt.Sprintf("https://gitlab.%s/kubefirst/gitops.git", domain)
	log.Info().Msgf("git remote add gitlab at url", upstream)

	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "gitlab",
		URLs: []string{upstream},
	})
	if err != nil {
		log.Error().Msgf("Error creating remote repo:", err)
	}
	w, _ := repo.Worktree()

	os.RemoveAll(directory + "/terraform/base/.terraform")
	os.RemoveAll(directory + "/terraform/gitlab/.terraform")
	os.RemoveAll(directory + "/terraform/vault/.terraform")

	log.Info().Msgf("Committing new changes...")
	_ = gitClient.GitAddWithFilter(viper.GetString("cloud"), "gitops", w)

	_, err = w.Commit("setting new remote upstream to gitlab", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: "kubefirst-bot@kubefirst.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		log.Panic().Msgf("error committing changes %s", err)
	}

	log.Info().Msg("setting auth...")
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
		log.Panic().Msgf("error pushing to remote %s", err)
	}

}

// AwaitHost - Await for a Host to be avialable, it wait for 200 cycles.
// Prefer to use `AwaitHostNTimes` as it provide more control
func AwaitHost(appName string, dryRun bool) {
	log.Info().Msg("AwaitHost called")
	AwaitHostNTimes(appName, dryRun, 200)
}

// AwaitHostNTimes - Wait for a Host to be responsive
// - To return 200
// - To return true if host is ready, or false if dont.
// - Supports to pass numbr of cycles to test
func AwaitHostNTimes(appName string, dryRun bool, times int) bool {
	log.Info().Msg("AwaitHostNTimes called")
	if dryRun {
		log.Printf("[#99] Dry-run mode, AwaitHost skipped.")
		return true
	}
	max := times
	hostReady := false
	for i := 0; i < max; i++ {
		hostedZoneName := viper.GetString("aws.hostedzonename")
		resp, _ := http.Get(fmt.Sprintf("https://%s.%s", appName, hostedZoneName))
		if resp != nil && resp.StatusCode == 200 {
			log.Printf("%s host resolved, 30 second grace period required...", appName)
			time.Sleep(time.Second * 30)
			i = max
			hostReady = true
			return hostReady
		} else {
			log.Printf("%s host not resolved, sleeping 10s", appName)
			time.Sleep(time.Second * 10)
		}
	}
	return hostReady
}

// ProduceGitlabTokens - Produce Gitlab token from argoCD secret
func ProduceGitlabTokens(dryRun bool) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, ProduceGitlabTokens skipped.")
		return
	}
	//TODO: Should this step be skipped if already executed?
	clientset, err := k8s.GetClientSet(dryRun)
	if err != nil {
		log.Panic().Msg(err.Error())
	}
	log.Info().Msg("discovering gitlab toolbox pod")
	time.Sleep(30 * time.Second)
	// todo: move it to config
	argocd.ArgocdSecretClient = clientset.CoreV1().Secrets("argocd")

	argocdPassword := k8s.GetSecretValue(argocd.ArgocdSecretClient, "argocd-initial-admin-secret", "password")
	if argocdPassword == "" {
		log.Panic().Msg("Missing argocdPassword")
	}

	viper.Set("argocd.admin.password", argocdPassword)
	viper.WriteConfig()

	log.Info().Msg("discovering gitlab toolbox pod")

	gitlabPodClient := clientset.CoreV1().Pods("gitlab")
	gitlabPodName := k8s.GetPodNameByLabel(gitlabPodClient, "app=toolbox")

	k8s.GitlabSecretClient = clientset.CoreV1().Secrets("gitlab")
	secrets, err := k8s.GitlabSecretClient.List(context.TODO(), metaV1.ListOptions{})

	var gitlabRootPasswordSecretName string

	for _, secret := range secrets.Items {
		if strings.Contains(secret.Name, "initial-root-password") {
			gitlabRootPasswordSecretName = secret.Name
			log.Info().Msgf("gitlab initial root password secret name: %s", gitlabRootPasswordSecretName)
		}
	}
	gitlabRootPassword := k8s.GetSecretValue(k8s.GitlabSecretClient, gitlabRootPasswordSecretName, "password")
	if gitlabRootPassword == "" {
		log.Panic().Msg("Missing gitlabRootPassword")
	}
	viper.Set("gitlab.podname", gitlabPodName)
	viper.Set("gitlab.root.password", gitlabRootPassword)
	viper.WriteConfig()

	gitlabToken := viper.GetString("gitlab.token")

	if gitlabToken == "" {

		log.Info().Msg("generating gitlab personal access token")
		GitlabGeneratePersonalAccessToken(gitlabPodName)

	}

	gitlabRunnerToken := viper.GetString("gitlab.runnertoken")

	if gitlabRunnerToken == "" {

		log.Info().Msg("getting gitlab runner token")
		gitlabRunnerRegistrationToken := k8s.GetSecretValue(k8s.GitlabSecretClient, "gitlab-gitlab-runner-secret", "runner-registration-token")
		if gitlabRunnerRegistrationToken == "" {
			log.Panic().Msg("Missing gitlabRunnerRegistrationToken")
		}
		viper.Set("gitlab.runnertoken", gitlabRunnerRegistrationToken)
		viper.WriteConfig()
	}

}

func ApplyGitlabTerraform(dryRun bool, directory string) {

	config := configs.ReadConfig()

	if !viper.GetBool("create.terraformapplied.gitlab") {
		log.Info().Msg("Executing applyGitlabTerraform")
		if dryRun {
			log.Printf("[#99] Dry-run mode, applyGitlabTerraform skipped.")
			return
		}
		//* AWS_SDK_LOAD_CONFIG=1
		//* https://registry.terraform.io/providers/hashicorp/aws/2.34.0/docs#shared-credentials-file
		envs := map[string]string{}
		envs["AWS_SDK_LOAD_CONFIG"] = "1"

		aws.ProfileInjection(&envs)

		// Prepare for terraform gitlab execution
		envs["GITLAB_TOKEN"] = viper.GetString("gitlab.token")
		envs["GITLAB_BASE_URL"] = viper.GetString("gitlab.local.service")

		directory = fmt.Sprintf("%s/gitops/terraform/gitlab", config.K1FolderPath)
		err := os.Chdir(directory)
		if err != nil {
			log.Panic().Msg("error: could not change directory to " + directory)
		}
		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init")
		if err != nil {
			log.Panic().Msgf("error: terraform init for gitlab failed %s", err)
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "apply", "-auto-approve")
		if err != nil {
			log.Panic().Msgf("error: terraform apply for gitlab failed %s", err)
		}
		os.RemoveAll(fmt.Sprintf("%s/.terraform", directory))
		viper.Set("create.terraformapplied.gitlab", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("Skipping: applyGitlabTerraform")
	}
}

func GitlabKeyUpload(dryRun bool) {

	// upload ssh public key
	if !viper.GetBool("gitlab.keyuploaded") {
		log.Info().Msg("Executing GitlabKeyUpload")
		log.Info().Msg("uploading ssh public key for gitlab user")
		if dryRun {
			log.Printf("[#99] Dry-run mode, GitlabKeyUpload skipped.")
			return
		}

		os.Setenv("AWS_SDK_LOAD_CONFIG", "1")

		log.Info().Msg("uploading ssh public key to gitlab")
		gitlabToken := viper.GetString("gitlab.token")
		data := url.Values{
			"title": {"kubefirst"},
			"key":   {viper.GetString("botpublickey")},
		}

		time.Sleep(10 * time.Second) // todo, build in a retry

		gitlabUrlBase := viper.GetString("gitlab.local.service")

		resp, err := http.PostForm(gitlabUrlBase+"/api/v4/user/keys?private_token="+gitlabToken, data)
		if err != nil {
			log.Panic().Msgf("%s", err)
		}
		var res map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&res)
		log.Info().Msgf("%s", res)
		log.Info().Msg("ssh public key uploaded to gitlab")
		viper.Set("gitlab.keyuploaded", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("Skipping: GitlabKeyUpload")
		log.Info().Msg("ssh public key already uploaded to gitlab")
	}
}

func DestroyGitlabTerraform(skipGitlabTerraform bool) {
	config := configs.ReadConfig()
	envs := map[string]string{}

	aws.ProfileInjection(&envs)

	envs["AWS_REGION"] = viper.GetString("aws.region")
	envs["AWS_ACCOUNT_ID"] = viper.GetString("aws.accountid")
	envs["HOSTED_ZONE_NAME"] = viper.GetString("aws.hostedzonename")
	envs["GITLAB_TOKEN"] = viper.GetString("gitlab.token")

	envs["TF_VAR_aws_account_id"] = viper.GetString("aws.accountid")
	envs["TF_VAR_aws_region"] = viper.GetString("aws.region")
	envs["TF_VAR_hosted_zone_name"] = viper.GetString("aws.hostedzonename")

	directory := fmt.Sprintf("%s/gitops/terraform/gitlab", config.K1FolderPath)
	err := os.Chdir(directory)
	if err != nil {
		log.Panic().Msgf("error: could not change directory to %s ", directory)
	}

	envs["GITLAB_BASE_URL"] = fmt.Sprintf("https://gitlab.%s", viper.GetString("aws.hostedzonename"))

	if !skipGitlabTerraform && viper.GetString("gitprovider") == "gitlab" {
		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "init")
		if err != nil {
			log.Panic().Msgf("failed to terraform init gitlab %s", err)
		}

		err = pkg.ExecShellWithVars(envs, config.TerraformClientPath, "destroy", "-auto-approve")
		if err != nil {
			log.Panic().Msgf("failed to terraform destroy gitlab %s", err)
		}

		viper.Set("destroy.terraformdestroy.gitlab", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("skip:  DestroyGitlabTerraform")
	}
}

func ChangeRegistryToGitLab(dryRun bool) {
	config := configs.ReadConfig()

	if dryRun {
		log.Printf("[#99] Dry-run mode, ChangeRegistryToGitLab skipped.")
		return
	}

	if !viper.GetBool("gitlab.changeregistry.gitlab") {
		type ArgocdGitCreds struct {
			PersonalAccessToken string
			URL                 string
			FullURL             string
		}

		pat := viper.GetString("gitlab.token")
		url := fmt.Sprintf("https://gitlab.%s/kubefirst/", viper.GetString("aws.hostedzonename"))
		fullurl := fmt.Sprintf("https://gitlab.%s/kubefirst/gitops.git", viper.GetString("aws.hostedzonename"))

		creds := ArgocdGitCreds{PersonalAccessToken: pat, URL: url, FullURL: fullurl}

		clientset, err := k8s.GetClientSet(dryRun)
		if err != nil {
			log.Panic().Msg("error getting kubeconfig for clientset")
		}
		argocd.ArgocdSecretClient = clientset.CoreV1().Secrets("argocd")

		argocdRepositoryAccessTokenSecret := &v1.Secret{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      "creds-gitlab",
				Namespace: "argocd",
				Labels: map[string]string{
					"argocd.argoproj.io/secret-type": "repo-creds",
				},
				Annotations: map[string]string{
					"managed-by": "argocd.argoproj.io",
				},
			},
			Data: map[string][]byte{
				"password": []byte(creds.PersonalAccessToken),
				"url":      []byte(creds.URL),
				"username": []byte("root"),
			},
			Type: "Opaque",
		}

		_ = argocd.ArgocdSecretClient.Delete(context.TODO(), "creds-gitlab", metaV1.DeleteOptions{})
		_, err = argocd.ArgocdSecretClient.Create(context.TODO(), argocdRepositoryAccessTokenSecret, metaV1.CreateOptions{})
		if err != nil {
			log.Panic().Msgf("error creating argocd repository credentials template %s", err)
		}

		argocdRepoSecret := &v1.Secret{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      "repo-gitlab",
				Namespace: "argocd",
				Labels: map[string]string{
					"argocd.argoproj.io/secret-type": "repository",
				},
				Annotations: map[string]string{
					"managed-by": "argocd.argoproj.io",
				},
			},
			Data: map[string][]byte{
				"project": []byte("default"),
				"type":    []byte("git"),
				"url":     []byte(creds.FullURL),
			},
			Type: "Opaque",
		}
		_ = argocd.ArgocdSecretClient.Delete(context.TODO(), "repo-gitlab", metaV1.DeleteOptions{})
		_, err = argocd.ArgocdSecretClient.Create(context.TODO(), argocdRepoSecret, metaV1.CreateOptions{})
		if err != nil {
			log.Panic().Msgf("error creating argocd repository connection secret %s", err)
		}

		// curl -X 'DELETE' \
		// 'https://$ARGO_ADDRESS/api/v1/applications/registry?cascade=false' \
		// -H 'accept: application/json'

		_, _, err = pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "apply", "-f", fmt.Sprintf("%s/gitops/components/gitlab/argocd-adopts-gitlab.yaml", config.K1FolderPath))
		if err != nil {
			log.Panic().Msgf("failed to call execute kubectl apply of argocd patch to adopt gitlab: %s", err)
		}
		viper.Set("gitlab.changeregistry.gitlab", true)
		viper.WriteConfig()
	}

}

func HydrateGitlabMetaphorRepo(dryRun bool) {
	cfg := configs.ReadConfig()
	//TODO: Should this be skipped if already executed?
	if !viper.GetBool("create.gitlabmetaphor.cloned") {
		if dryRun {
			log.Printf("[#99] Dry-run mode, hydrateGitlabMetaphorRepo skipped.")
			return
		}

		metaphorTemplateDir := fmt.Sprintf("%s/metaphor", cfg.K1FolderPath)

		url := "https://github.com/kubefirst/metaphor-template"

		metaphorTemplateRepo, err := git.PlainClone(metaphorTemplateDir, false, &git.CloneOptions{
			URL: url,
		})
		if err != nil {
			log.Panic().Msg("error cloning metaphor-template repo")
		}
		viper.Set("create.gitlabmetaphor.cloned", true)

		pkg.Detokenize(metaphorTemplateDir)

		viper.Set("create.gitlabmetaphor.detokenized", true)

		// todo make global
		gitlabURL := fmt.Sprintf("https://gitlab.%s", viper.GetString("aws.hostedzonename"))
		log.Info().Msgf("gitClient remote add origin %s", gitlabURL)
		_, err = metaphorTemplateRepo.CreateRemote(&config.RemoteConfig{
			Name: "gitlab",
			URLs: []string{fmt.Sprintf("%s/kubefirst/metaphor.gitClient", gitlabURL)},
		})

		w, _ := metaphorTemplateRepo.Worktree()

		log.Info().Msg("Committing detokenized metaphor content")
		_ = gitClient.GitAddWithFilter(viper.GetString("cloud"), "metaphor", w)
		w.Commit("setting new remote upstream to gitlab", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "kubefirst-bot",
				Email: "kubefirst-bot@kubefirst.com",
				When:  time.Now(),
			},
		})

		err = metaphorTemplateRepo.Push(&git.PushOptions{
			RemoteName: "gitlab",
			Auth: &gitHttp.BasicAuth{
				Username: "root",
				Password: viper.GetString("gitlab.token"),
			},
		})
		if err != nil {
			log.Panic().Msgf("error pushing detokenized metaphor repository to remote at %s" + gitlabURL)
		}

		viper.Set("create.gitlabmetaphor.pushed", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("Skipping: hydrateGitlabMetaphorRepo")
	}

}

// refactor: review it
func PushGitRepo(dryRun bool, config *configs.Config, gitOrigin, repoName string) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, PushGitRepo skipped.")
		return
	}
	repoDir := fmt.Sprintf("%s/%s", config.K1FolderPath, repoName)
	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		log.Panic().Msgf("error opening repo %s: %s", repoName, err)
	}

	// todo - fix opts := &git.PushOptions{uniqe, stuff} .Push(opts) ?
	if gitOrigin == "soft" {
		pkg.Detokenize(repoDir)
		os.RemoveAll(repoDir + "/terraform/base/.terraform")
		os.RemoveAll(repoDir + "/terraform/gitlab/.terraform")
		os.RemoveAll(repoDir + "/terraform/vault/.terraform")
		os.Remove(repoDir + "/terraform/base/.terraform.lock.hcl")
		os.Remove(repoDir + "/terraform/vault/.terraform.lock.hcl")
		os.Remove(repoDir + "/terraform/users/.terraform.lock.hcl")
		os.Remove(repoDir + "/terraform/gitlab/.terraform.lock.hcl")
		CommitToRepo(repo, repoName)
		auth, _ := internalSSH.PublicKey()

		auth.HostKeyCallback = ssh.InsecureIgnoreHostKey()

		err = repo.Push(&git.PushOptions{
			RemoteName: gitOrigin,
			Auth:       auth,
		})
		if err != nil {
			log.Panic().Msgf("error pushing detokenized %s repository to remote at %s", repoName, gitOrigin)
		}
		log.Printf("successfully pushed %s to soft-serve", repoName)
		//TODO: Remove me
		cmd := exec.Command("cp", "-r", repoDir, repoDir+"-"+gitOrigin)
		err := cmd.Run()
		log.Info().Msgf("%s", err)
	}

	if gitOrigin == "gitlab" {
		registryFileContent := `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: argocd-components
  namespace: argocd
  annotations:
    argocd.argoproj.io/sync-wave: "100"
spec:
  project: default
  source:
    repoURL: ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops
    path: components/argocd-gitlab
    targetRevision: HEAD
  destination:
    server: https://kubernetes.default.svc
    namespace: argocd
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
    retry:
      limit: 5
      backoff:
        duration: 5s
        maxDuration: 5m0s
        factor: 2`
		file, err := os.Create(fmt.Sprintf("%s/gitops/registry/argocd.yaml", config.K1FolderPath))
		if err != nil {
			log.Info().Msgf("%s", err)
		}
		_, err = file.WriteString(registryFileContent)
		if err != nil {
			log.Info().Msgf("%s", err)
		}
		file.Close()

		pkg.Detokenize(repoDir)
		os.RemoveAll(repoDir + "/terraform/base/.terraform")
		os.RemoveAll(repoDir + "/terraform/gitlab/.terraform")
		os.RemoveAll(repoDir + "/terraform/vault/.terraform")
		os.Remove(repoDir + "/terraform/base/.terraform.lock.hcl")
		os.Remove(repoDir + "/terraform/vault/.terraform.lock.hcl")
		os.Remove(repoDir + "/terraform/users/.terraform.lock.hcl")
		os.Remove(repoDir + "/terraform/gitlab/.terraform.lock.hcl")

		CommitToRepo(repo, repoName)
		auth := &gitHttp.BasicAuth{
			Username: "root",
			Password: viper.GetString("gitlab.token"),
		}
		err = repo.Push(&git.PushOptions{
			RemoteName: gitOrigin,
			Auth:       auth,
		})
		if err != nil {
			log.Info().Msgf("%s", err)
			log.Panic().Msgf("error pushing detokenized %s repository to remote at %s", repoName, gitOrigin)
		}
		log.Printf("successfully pushed %s to gitlab", repoName)
		//cmd := exec.Command("cp", "-r", repoDir, repoDir+"-"+gitOrigin)
		//err = cmd.Run()
		//log.Info().Msg(err)
	}

	viper.Set(fmt.Sprintf("create.repos.%s.%s.pushed", gitOrigin, repoName), true)
	viper.WriteConfig()
}

// refactor: review it
func CommitToRepo(repo *git.Repository, repoName string) {
	w, _ := repo.Worktree()

	log.Info().Msg(fmt.Sprintf("committing detokenized %s kms key id", repoName))

	_ = gitClient.GitAddWithFilter(viper.GetString("cloud"), repoName, w)

	w.Commit(fmt.Sprintf("committing detokenized %s kms key id", repoName), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: "kubefirst-bot@kubefirst.com",
			When:  time.Now(),
		},
	})
}
