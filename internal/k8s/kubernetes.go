/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kubefirst/nebulous/configs"
	"github.com/spf13/viper"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"
	"log"
	"os"
	"os/exec"
	"syscall"
)

var vaultRootToken string
var gitlabToolboxPodName string

// API client for managing secrets & pods
var GitlabSecretClient coreV1Types.SecretInterface
var VaultSecretClient coreV1Types.SecretInterface
var ArgocdSecretClient coreV1Types.SecretInterface
var GitlabPodsClient coreV1Types.PodInterface

func GetPodNameByLabel(gitlabPodsClient coreV1Types.PodInterface, label string) string {
	pods, err := gitlabPodsClient.List(context.TODO(), metaV1.ListOptions{LabelSelector: fmt.Sprintf("app=%s", label)})
	if err != nil {
		fmt.Println(err)
	}

	gitlabToolboxPodName = pods.Items[0].Name

	return gitlabToolboxPodName
}

func getVaultRootToken(vaultSecretClient coreV1Types.SecretInterface) string {
	name := "vault-unseal-keys"
	log.Printf("Reading secret %s\n", name)
	secret, err := vaultSecretClient.Get(context.TODO(), name, metaV1.GetOptions{})

	if err != nil {
		log.Panic(err.Error())
	}

	var jsonData map[string]interface{}

	for _, value := range secret.Data {
		if err := json.Unmarshal(value, &jsonData); err != nil {
			log.Panic(err)
		}
		vaultRootToken = jsonData["root_token"].(string)
	}
	return vaultRootToken
}

func GetSecretValue(k8sClient coreV1Types.SecretInterface, secretName, key string) string {
	secret, err := k8sClient.Get(context.TODO(), secretName, metaV1.GetOptions{})
	if err != nil {
		log.Println(fmt.Sprintf("error getting key: %s from secret: %s", key, secretName), err)
	}
	return string(secret.Data[key])
}

func DeleteRegistryApplication(skipDeleteRegistryApplication bool) {
	config := configs.ReadConfig()
	if !skipDeleteRegistryApplication {
		log.Println("starting port forward to argocd server and deleting registry")
		kPortForward := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "port-forward", "svc/argocd-server", "8080:8080")
		kPortForward.Stdout = os.Stdout
		kPortForward.Stderr = os.Stderr
		err := kPortForward.Start()
		defer kPortForward.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Panicf("error: failed to port-forward to argocd %s", err)
		}

		url := "https://localhost:8080/api/v1/applications/registry"
		argoCdAppSync := exec.Command("curl", "-k", "-vL", "-X", "DELETE", url, "-H", fmt.Sprintf("Authorization: Bearer %s", viper.GetString("argocd.admin.apitoken")))
		argoCdAppSync.Stdout = os.Stdout
		argoCdAppSync.Stderr = os.Stderr
		err = argoCdAppSync.Run()
		if err != nil {
			log.Panicf("error: curl appSync failed failed %s", err)
		}
		log.Println("deleting argocd application registry")
	} else {
		log.Println("skip:  deleteRegistryApplication")
	}
}
