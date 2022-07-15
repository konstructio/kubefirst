package ssl

import (
	"context"
	"fmt"
	"github.com/itchyny/gojq"
	"github.com/kubefirst/kubefirst/cmd"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
)

func BackupCertificates() {
	//config := configs.ReadConfig()
	//
	//secrets, errF := os.Create("/tmp/secrets.yaml")
	//if errF != nil {
	//	panic(errF)
	//}
	//defer secrets.Close()
	//
	//kGetAllSecrets := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "secrets", "-A")
	//kGetAllSecrets.Stdout = secrets
	//err := kGetAllSecrets.Run()
	//
	//if err != nil {
	//	log.Println("Secrets get successful: " + err.Error())
	//} else {
	//	log.Println("Error getting secretsL: " + err.Error())
	//}
	ctx := context.Background()
	config := ctrl.GetConfigOrDie()
	dynamic := dynamic.NewForConfigOrDie(config)

	namespace := "default"
	query := ".metadata.labels[\"app.kubernetes.io/managed-by\"] == \"Helm\""

	items, err := GetResourcesByJq(dynamic, ctx, "apps", "v1", "deployments", namespace, query)
	if err != nil {
		fmt.Println(err)
	} else {
		for _, item := range items {
			fmt.Printf("%+v\n", item)
		}
	}
}

func GetResourcesByJq(dynamic dynamic.Interface, ctx context.Context, group string,
	version string, resource string, namespace string, jq string) (
	[]unstructured.Unstructured, error) {

	resources := make([]unstructured.Unstructured, 0)

	query, err := gojq.Parse(jq)
	if err != nil {
		return nil, err
	}

	items, err := cmd.GetResourcesDynamically(dynamic, ctx, group, version, resource, namespace)
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

func RestoreCertificates() {

}
