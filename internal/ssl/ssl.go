package ssl

import (
	"context"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

// GetBackupCertificates create a backup of Certificates on AWS S3 in yaml files
func GetBackupCertificates(namespaces []string) ([]string, error) {
	config := configs.ReadConfig()

	k8sConfig, err := clientcmd.BuildConfigFromFlags("", config.KubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error getting k8sClient %s", err)
	}

	k8sClient := dynamic.NewForConfigOrDie(k8sConfig)
	var files []string
	for _, namespace := range namespaces {
		// items, err := k8s.GetResourcesDynamically(k8sClient, context.TODO(),
		// 	"cert-manager.io", "v1", "certificates", namespace)
		// if err != nil {
		// 	return nil, fmt.Errorf("error getting resources from k8s: %s", err)
		// }

		k8sResourceTypes := []string{
			"certificates",
			//"issuer",
			//"clusterissuer",
		}

		items, err := k8s.GetResourcesDynamically(k8sClient, context.TODO(),
			"cert-manager.io", "v1", k8sResourceTypes, namespace)
		if err != nil {
			return nil, fmt.Errorf("error getting resources from k8s: %s", err)
		}

		for _, item := range items {
			jsonObj, err := item.MarshalJSON()
			if err != nil {
				return nil, fmt.Errorf("error converting object on json: %s", err)
			}
			//yamlObj, err := yaml.JSONToYAML(jsonObj)
			yamlObj, err := yaml.JSONToYAML(jsonObj)
			if err != nil {
				return nil, fmt.Errorf("error converting object from json to yaml: %s", err)
			}
			fmt.Println("---debug---")
			fmt.Println(string(yamlObj))
			fmt.Println("---debug---")

			// todo: uncomment after unblocking the code above
			//fileName := fmt.Sprintf("%s.%s", item.GetName(), "yaml")
			//err = pkg.CreateFile(fileName, yamlObj)
			//if err != nil {
			//	return nil, err
			//}
			//files = append(files, fileName)
		}
	}

	return files, nil
}
