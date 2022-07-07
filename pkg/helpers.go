package pkg

import (
	"bytes"
	"context"
	b64 "encoding/base64"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/kubefirst/nebulous/configs"
	"github.com/kubefirst/nebulous/internal/k8s"
	"github.com/spf13/viper"
	"html/template"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Detokenize(path string) {

	err := filepath.Walk(path, DetokenizeDirectory)
	if err != nil {
		panic(err)
	}
}

func DetokenizeDirectory(path string, fi os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if fi.IsDir() {
		return nil //
	}

	if strings.Contains(path, ".gitClient") || strings.Contains(path, ".terraform") {
		return nil
	}

	matched, err := filepath.Match("*", fi.Name())

	if err != nil {
		panic(err)
	}

	if matched {
		read, err := ioutil.ReadFile(path)
		if err != nil {
			panic(err)
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
		domainId := viper.GetString("aws.domainid")
		hostedzonename := viper.GetString("aws.hostedzonename")
		bucketStateStore := viper.GetString("bucket.state-store.name")
		bucketArgoArtifacts := viper.GetString("bucket.argo-artifacts.name")
		bucketGitlabBackup := viper.GetString("bucket.gitlab-backup.name")
		bucketChartmuseum := viper.GetString("bucket.chartmuseum.name")
		region := viper.GetString("aws.region")
		adminEmail := viper.GetString("adminemail")
		awsAccountId := viper.GetString("aws.accountid")
		kmsKeyId := viper.GetString("vault.kmskeyid")

		newContents = strings.Replace(newContents, "<SOFT_SERVE_INITIAL_ADMIN_PUBLIC_KEY>", strings.TrimSpace(botPublicKey), -1)
		newContents = strings.Replace(newContents, "<TF_STATE_BUCKET>", bucketStateStore, -1)
		newContents = strings.Replace(newContents, "<ARGO_ARTIFACT_BUCKET>", bucketArgoArtifacts, -1)
		newContents = strings.Replace(newContents, "<GITLAB_BACKUP_BUCKET>", bucketGitlabBackup, -1)
		newContents = strings.Replace(newContents, "<CHARTMUSEUM_BUCKET>", bucketChartmuseum, -1)
		newContents = strings.Replace(newContents, "<AWS_HOSTED_ZONE_ID>", domainId, -1)
		newContents = strings.Replace(newContents, "<AWS_HOSTED_ZONE_NAME>", hostedzonename, -1)
		newContents = strings.Replace(newContents, "<AWS_DEFAULT_REGION>", region, -1)
		newContents = strings.Replace(newContents, "<EMAIL_ADDRESS>", adminEmail, -1)
		newContents = strings.Replace(newContents, "<AWS_ACCOUNT_ID>", awsAccountId, -1)
		if kmsKeyId != "" {
			newContents = strings.Replace(newContents, "<KMS_KEY_ID>", kmsKeyId, -1)
		}

		if viper.GetBool("create.terraformapplied.gitlab") {
			newContents = strings.Replace(newContents, "<AWS_HOSTED_ZONE_NAME>", hostedzonename, -1)
			newContents = strings.Replace(newContents, "<AWS_DEFAULT_REGION>", region, -1)
			newContents = strings.Replace(newContents, "<AWS_ACCOUNT_ID>", awsAccountId, -1)
		}

		err = ioutil.WriteFile(path, []byte(newContents), 0)
		if err != nil {
			panic(err)
		}

	}

	return nil
}

func ChangeRegistryToGitLab() {
	config := configs.ReadConfig()
	if !viper.GetBool("gitlab.registry") {
		if config.DryRun {
			log.Printf("[#99] Dry-run mode, ChangeRegistryToGitLab skipped.")
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
		k8sConfig, err := clientcmd.BuildConfigFromFlags("", config.KubeConfigPath)
		if err != nil {
			log.Panicf("error getting client from kubeconfig")
		}
		clientset, err := kubernetes.NewForConfig(k8sConfig)
		if err != nil {
			log.Panicf("error getting kubeconfig for clientset")
		}
		k8s.ArgocdSecretClient = clientset.CoreV1().Secrets("argocd")

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

		_, err = k8s.ArgocdSecretClient.Create(context.TODO(), argocdRepositoryAccessTokenSecret, metaV1.CreateOptions{})
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

		_, err = k8s.ArgocdSecretClient.Create(context.TODO(), argocdRepositoryAccessTokenSecret, metaV1.CreateOptions{})
		if err != nil {
			log.Panicf("error creating argocd repository connection secret %s", err)
		}

		k := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "apply", "-f", fmt.Sprintf("%s/.kubefirst/gitops/components/gitlab/argocd-adopts-gitlab.yaml", config.HomePath))
		k.Stdout = os.Stdout
		k.Stderr = os.Stderr
		err = k.Run()
		if err != nil {
			log.Panicf("failed to call execute kubectl apply of argocd patch to adopt gitlab: %s", err)
		}

		viper.Set("gitlab.registry", true)
		viper.WriteConfig()
	} else {
		log.Println("Skipping: ChangeRegistryToGitLab")
	}
}
