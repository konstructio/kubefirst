package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/spf13/viper"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

	if !skipDeleteRegistryApplication {

		log.Println("refreshing argocd session token")
		argocd.GetArgocdAuthToken(false)

		url := "https://localhost:8080/api/v1/applications/registry"
		argoCdAppSync := exec.Command("curl", "-k", "-vL", "-X", "DELETE", url, "-H", fmt.Sprintf("Authorization: Bearer %s", viper.GetString("argocd.admin.apitoken")))
		argoCdAppSync.Stdout = os.Stdout
		argoCdAppSync.Stderr = os.Stderr
		err := argoCdAppSync.Run()
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
	resources []string,
	namespace string,
) ([]unstructured.UnstructuredList, error) {

	var listTypes []unstructured.UnstructuredList

	for _, resource := range resources {

		resourceId := schema.GroupVersionResource{
			Group:    group,
			Version:  version,
			Resource: resource,
		}

		resourceList, err := dynamic.Resource(resourceId).Namespace(namespace).List(ctx, metaV1.ListOptions{})
		if err != nil {
			return nil, err
		}
		listTypes = append(listTypes, *resourceList)
	}

	return listTypes, nil
}
