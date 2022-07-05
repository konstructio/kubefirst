package kubefirst

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"
)

var vaultSecretClient coreV1Types.SecretInterface

func GetPodNameByLabel(gitlabPodsClient coreV1Types.PodInterface, label string) string {

	var gitlabToolboxPodName string

	pods, err := gitlabPodsClient.List(context.TODO(), metaV1.ListOptions{LabelSelector: fmt.Sprintf("app=%s", label)})
	if err != nil {
		panic(fmt.Sprintf("error: failed to list using gitlab pods client %s", err))
	}

	gitlabToolboxPodName = pods.Items[0].Name

	return gitlabToolboxPodName
}

func GetVaultRootToken(vaultSecretClient coreV1Types.SecretInterface) string {
	var vaultRootToken string
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
