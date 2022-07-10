/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/viper"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"
)

var vaultRootToken string
var gitlabToolboxPodName string

// API client for managing secrets & pods
var gitlabSecretClient coreV1Types.SecretInterface
var vaultSecretClient coreV1Types.SecretInterface
var argocdSecretClient coreV1Types.SecretInterface
var gitlabPodsClient coreV1Types.PodInterface

func getPodNameByLabel(gitlabPodsClient coreV1Types.PodInterface, label string) string {
	pods, err := gitlabPodsClient.List(context.TODO(), metaV1.ListOptions{LabelSelector: fmt.Sprintf("app=%s", label)})
	if err != nil {
		fmt.Println(err)
	}

	gitlabToolboxPodName = pods.Items[0].Name

	return gitlabToolboxPodName
}

func waitForVaultUnseal() {
	vaultReady := viper.GetBool("create.vault.ready")
	if !vaultReady {
		var output bytes.Buffer
		// todo - add a viper.GetBool() check to the beginning of this function
		// todo write in golang? see here -> https://github.com/bcreane/k8sutils/blob/master/utils.go
		k := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "vault", "wait", "--for=condition=ready", "pod", "-l", "vault-initialized=true", "--timeout=300s")
		k.Stdout = &output
		k.Stderr = os.Stderr
		err := k.Run()
		if err != nil {
			log.Panicf("failed to execute kubectl wait for vault pods with label vault-initialized=true: %s \n%s", output, err)
		}
		log.Printf("the output is: %s", output.String())
	} else {
		log.Println("vault is ready")
	}

}

func waitForGitlab() {
	var output bytes.Buffer
	// todo - add a viper.GetBool() check to the beginning of this function
	// todo write in golang? see here -> https://github.com/bcreane/k8sutils/blob/master/utils.go
	k := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "gitlab", "wait", "--for=condition=ready", "pod", "-l", "app=webservice", "--timeout=300s")
	k.Stdout = &output
	k.Stderr = os.Stderr
	err := k.Run()
	if err != nil {
		log.Panicf("failed to execute kubectl wait for gitlab pods with label app=webservice: %s \n%s", output, err)
	}
	log.Printf("the output is: %s", output.String())
}

func createVaultConfiguredSecret() {
	if !viper.GetBool("vault.configuredsecret") {
		var output bytes.Buffer
		// todo - https://github.com/bcreane/k8sutils/blob/master/utils.go
		// kubectl create secret generic vault-configured --from-literal=isConfigured=true
		// the purpose of this command is to let the vault-unseal Job running in kuberenetes know that external secrets store should be able to connect to the configured vault
		k := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "vault", "create", "secret", "generic", "vault-configured", "--from-literal=isConfigured=true")
		k.Stdout = &output
		k.Stderr = os.Stderr
		err := k.Run()
		if err != nil {
			log.Panicf("failed to create secret for vault-configured: %s", err)
		}
		log.Println("the secret create output is: %s", output.String())

		viper.Set("vault.configuredsecret", true)
		viper.WriteConfig()
	} else {
		log.Println("vault secret already created")
	}
}

func getVaultRootToken(vaultSecretClient coreV1Types.SecretInterface) string {
	name := "vault-unseal-keys"
	log.Printf("Reading secret %s\n", name)
	secret, err := vaultSecretClient.Get(context.TODO(), name, metaV1.GetOptions{})

	if err != nil {
		panic(err.Error())
	}

	var jsonData map[string]interface{}

	for _, value := range secret.Data {
		if err := json.Unmarshal(value, &jsonData); err != nil {
			panic(err)
		}
		vaultRootToken = jsonData["root_token"].(string)
	}
	return vaultRootToken
}

func getSecretValue(k8sClient coreV1Types.SecretInterface, secretName, key string) string {
	secret, err := k8sClient.Get(context.TODO(), secretName, metaV1.GetOptions{})
	if err != nil {
		log.Println(fmt.Sprintf("error getting key: %s from secret: %s", key, secretName), err)
	}
	return string(secret.Data[key])
}

func waitForNamespaceandPods(namespace, podLabel string) {
	x := 50
	for i := 0; i < x; i++ {
		kGetNamespace := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", namespace, "get", fmt.Sprintf("namespace/%s", namespace))
		kGetNamespace.Stdout = os.Stdout
		kGetNamespace.Stderr = os.Stderr
		err := kGetNamespace.Run()
		if err != nil {
			log.Println(fmt.Sprintf("waiting for %s namespace to create ", namespace))
			time.Sleep(10 * time.Second)
		} else {
			log.Println(fmt.Sprintf("namespace %s found, continuing", namespace))
			time.Sleep(10 * time.Second)
			i = 51
		}
	}
	for i := 0; i < x; i++ {
		kGetPods := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", namespace, "get", "pods", "-l", podLabel)
		kGetPods.Stdout = os.Stdout
		kGetPods.Stderr = os.Stderr
		err := kGetPods.Run()
		if err != nil {
			log.Println(fmt.Sprintf("waiting for %s pods to create ", namespace))
			time.Sleep(10 * time.Second)
		} else {
			log.Println(fmt.Sprintf("%s pods found, continuing", namespace))
			time.Sleep(10 * time.Second)
			break
		}
	}
}
