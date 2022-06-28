/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

var vaultRootToken string
var gitlabToolboxPodName string

// API client for managing secrets & pods
var gitlabSecretClient coreV1Types.SecretInterface
var vaultSecretClient coreV1Types.SecretInterface
var argocdSecretClient coreV1Types.SecretInterface
var gitlabPodsClient coreV1Types.PodInterface

// kubectlCmd represents the kubectl command
var kubectlCmd = &cobra.Command{
	Use:   "kubectl",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		kubeconfig := os.Getenv("HOME") + "/.kube/config"
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
		//!
		// todo create an ecr registry token
		// todo create a kubernetes secret containing the object required
		// todo
		// todo

		cfg, err := awsConfig.LoadDefaultConfig(context.TODO())
		if err != nil {
			fmt.Println("failed to load configuration, error:", err)
		}
		// https://aws.github.io/aws-sdk-go-v2/docs/making-requests/#overriding-configuration
		ecrClient := ecr.NewFromConfig(cfg)

		token, err := ecrClient.GetAuthorizationToken(context.TODO(), &ecr.GetAuthorizationTokenInput{})
		if err != nil {
			fmt.Println("error getting ecr token: ", err.Error())
		}

		ecrAccessToken := *token.AuthorizationData[0].AuthorizationToken

		argocdSecretClient = clientset.CoreV1().Secrets("argocd")

		var argocdRepositoryAccessTokenSecret *v1.Secret

		argocdRepositoryAccessTokenSecretYaml := []byte(fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: argocd-ecr-access-token
type: Opaque
data:
  ecr-token: %s
    `, ecrAccessToken))

		err = yaml.Unmarshal(argocdRepositoryAccessTokenSecretYaml, &argocdRepositoryAccessTokenSecret)
		if err != nil {
			panic(err.Error())
		}

		_, err = argocdSecretClient.Create(context.TODO(), argocdRepositoryAccessTokenSecret, metaV1.CreateOptions{})
		if err != nil {
			panic(err)
		}
		fmt.Printf("Created secret argocd-ecr-access-token")
		//!

		// should these be more re-usable?
		//* requires already namespaced client
		// vaultSecretClient = clientset.CoreV1().Secrets("vault")
		// // vaultRootToken := getVaultRootToken(vaultSecretClient)

		// gitlabPodsClient = clientset.CoreV1().Pods("gitlab")
		// //* requires already namespaced client
		// // podName := getPodNameByLabel(gitlabPodsClient, "toolbox")
		// // fmt.Println(podName)

		// gitlabSecretClient = clientset.CoreV1().Secrets("gitlab")
		// secrets, err := gitlabSecretClient.List(context.TODO(), metaV1.ListOptions{})

		// var gitlabRootPasswordSecretName string

		// for _, secret := range secrets.Items {
		// 	if strings.Contains(secret.Name, "initial-root-password") {
		// 		gitlabRootPasswordSecretName = secret.Name
		// 		fmt.Println("gitlab root password secret name: ", gitlabRootPasswordSecretName)
		// 	}
		// }
		// gitlabRootPassword := getSecretValue(gitlabSecretClient, gitlabRootPasswordSecretName, "password")

		// fmt.Println("gitlab root password: ", gitlabRootPassword)

	},
}

func init() {
	nebulousCmd.AddCommand(kubectlCmd)
}

func getPodNameByLabel(gitlabPodsClient coreV1Types.PodInterface, label string) string {
	pods, err := gitlabPodsClient.List(context.TODO(), metaV1.ListOptions{LabelSelector: fmt.Sprintf("app=%s", label)})
	if err != nil {
		fmt.Println(err)
	}

	gitlabToolboxPodName = pods.Items[0].Name

	return gitlabToolboxPodName
}

func getVaultRootToken(vaultSecretClient coreV1Types.SecretInterface) string {
	name := "vault-unseal-keys"
	fmt.Printf("Reading secret %s\n", name)
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
		fmt.Println(fmt.Sprintf("error getting key: %s from secret: %s", key, secretName), err)
	}
	return string(secret.Data[key])
}
