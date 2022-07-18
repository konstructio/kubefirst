package ssl

import (
	"context"
	"fmt"
	"log"

	"github.com/kubefirst/kubefirst/configs"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

func BackupCertificates() {
	config := configs.ReadConfig()

	k8sClient, err := clientcmd.BuildConfigFromFlags("", config.KubeConfigPath)
	if err != nil {
		log.Panicf("error: getting k8sClient %s", err)
	}

	dynamic := dynamic.NewForConfigOrDie(k8sClient)

	namespace := "argo"

	items, err := GetResourcesDynamically(dynamic, context.TODO(),
		"cert-manager.io", "v1", "certificates", namespace)
	if err != nil {
		fmt.Println(err)
	} else {
		for _, item := range items {
			fmt.Printf("%+v\n", item)
		}
	}

}

func GetResourcesDynamically(dynamic dynamic.Interface, ctx context.Context,
	group string, version string, resource string, namespace string) (
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

func RestoreCertificates() {

}
