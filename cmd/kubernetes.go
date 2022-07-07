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
	var output bytes.Buffer
	// todo - add a viper.GetBool() check to the beginning of this function
	// todo write in golang? see here -> https://github.com/bcreane/k8sutils/blob/master/utils.go
	k := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "vault", "wait", "--for=condition=ready", "pod", "-l", "vault-sealed=false", "--timeout=120s")
	k.Stdout = &output
	k.Stderr = os.Stderr
	err := k.Run()
	if err != nil {
		log.Panicf("failed to execute kubectl wait for vault pods with label vault-sealed=false: %s \n%s", output, err)
	}
	log.Printf("the output is: %s", output.String())

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
