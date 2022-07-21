/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/itchyny/gojq"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"
)

var vaultRootToken string
var gitlabToolboxPodName string

// API client for managing secrets & pods
var GitlabSecretClient coreV1Types.SecretInterface
var VaultSecretClient coreV1Types.SecretInterface
var ArgocdSecretClient coreV1Types.SecretInterface

// var GitlabPodsClient coreV1Types.PodInterface

func GetPodNameByLabel(podsClient coreV1Types.PodInterface, label string) string {
	pods, err := podsClient.List(context.TODO(), metaV1.ListOptions{LabelSelector: label})
	if err != nil {
		log.Println(err)
	}

	gitlabToolboxPodName = pods.Items[0].Name

	return gitlabToolboxPodName
}

func DeletePodByName(podsClient coreV1Types.PodInterface, podName string) {
	err := podsClient.Delete(context.TODO(), podName, metaV1.DeleteOptions{})
	if err != nil {
		log.Println(err)
	}
}

// func CreateRepoSecret() {

// }

// func CreateCredentialsTemplateSecret() {

// }

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

	if !skipDeleteRegistryApplication {

		log.Println("refreshing argocd session token")
		argocd.GetArgocdAuthToken(false)

		url := "https://localhost:8080/api/v1/applications/registry"
		_, _, err := pkg.ExecShellReturnStrings("curl", "-k", "-vL", "-X", "DELETE", url, "-H", fmt.Sprintf("Authorization: Bearer %s", viper.GetString("argocd.admin.apitoken")))
		if err != nil {
			log.Panicf("error: delete registry applicatoin from argocd failed: %s", err)
		}
		log.Println("waiting for argocd deletion to complete")
		time.Sleep(300 * time.Second)
	} else {
		log.Println("skip:  deleteRegistryApplication")
	}
}

func GetResourcesDynamically(dynamic dynamic.Interface,
	ctx context.Context,
	group string,
	version string,
	resource string,
	namespace string) (
	[]unstructured.Unstructured, error) {

	resourceId := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}
	list, err := dynamic.Resource(resourceId).Namespace(namespace).
		List(ctx, metaV1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

func GetResourcesByJq(dynamic dynamic.Interface, ctx context.Context, group string,
	version string, resource string, namespace string, jq string) (
	[]unstructured.Unstructured, error) {

	resources := make([]unstructured.Unstructured, 0)

	query, err := gojq.Parse(jq)
	if err != nil {
		return nil, err
	}

	items, err := GetResourcesDynamically(dynamic, ctx, group, version, resource, namespace)
	if err != nil {
		return nil, err
	}

	for _, item := range items {

		// Convert object to raw JSON
		var rawJson interface{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &rawJson)
		if err != nil {
			return nil, err
		}

		// Evaluate jq against JSON
		iter := query.Run(rawJson)
		for {
			result, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok := result.(error); ok {
				if err != nil {
					return nil, err
				}
			} else {
				boolResult, ok := result.(bool)
				if !ok {
					fmt.Println("Query returned non-boolean value")
				} else if boolResult {
					resources = append(resources, item)
				}
			}
		}
	}
	return resources, nil
}
