package ssl

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

func getItemsToBackup(apiGroup string, apiVersion string, resourceType string, namespaces []string, jqQuery string) ([]string, error) {
	config := configs.ReadConfig()

	k8sConfig, err := clientcmd.BuildConfigFromFlags("", config.KubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error getting k8sClient %s", err)
	}

	k8sClient := dynamic.NewForConfigOrDie(k8sConfig)

	var files []string
	var items []unstructured.Unstructured
	for _, namespace := range namespaces {
		if len(jqQuery) > 0 {
			fmt.Println("getting resources and filtering using jq")
			items, err = k8s.GetResourcesByJq(k8sClient, context.TODO(), apiGroup, apiVersion, resourceType, namespace, jqQuery)
		} else {
			fmt.Println("getting resources")
			items, err = k8s.GetResourcesDynamically(k8sClient, context.TODO(), apiGroup, apiVersion, resourceType, namespace)
		}

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

			fileName := fmt.Sprintf("%s.%s", item.GetName(), "yaml")
			//TODO: test if kubeconfigpath is the correct value to write the files together another k1rst files
			fullFileName := filepath.Join(config.KubeConfigPath, fileName)
			err = pkg.CreateFile(fullFileName, yamlObj)
			if err != nil {
				return nil, err
			}
			files = append(files, fullFileName)
		}
	}

	return files, nil
}

//func GetBackupCertificates(apiGroup string, apiVersion string, resourceTypes []string, namespace string) ([]string, error) {
// GetBackupCertificates create a backup of Certificates on AWS S3 in yaml files
func GetBackupCertificates() (string, error) {

	fmt.Println("GetBackupCertificates called")
	bucketName := fmt.Sprintf("k1-%s", viper.GetString("aws.hostedzonename"))
	path := "cert-manager"
	aws.CreateBucket(false, bucketName)

	fmt.Println("getting certificates")
	namespaces := []string{"argo", "atlantis", "chartmuseum", "gitlab", "vault"}
	certificates, err := getItemsToBackup("cert-manager.io", "v1", "certificates", namespaces, "")
	if err != nil {
		log.Panic(err)
	}
	for _, cert := range certificates {
		fullPath := fmt.Sprintf("%s/cert-%s", path, cert)
		fmt.Println(fullPath)
		aws.UploadFile(bucketName, fullPath, cert)
	}

	fmt.Println("getting secrets")
	query := ".metadata.annotations[\"cert-manager.io/issuer-kind\"] == \"ClusterIssuer\""
	secrets, err := getItemsToBackup("", "v1", "secrets", namespaces, query)
	if err != nil {
		log.Panic(err)
	}
	for _, secret := range secrets {
		fullPath := fmt.Sprintf("%s/secret-%s", path, secret)
		fmt.Println(fullPath)
		aws.UploadFile(bucketName, fullPath, secret)
	}

	emptyNS := []string{""}
	fmt.Println("getting clusterissuers")
	clusterIssuers, err := getItemsToBackup("cert-manager.io", "v1", "clusterissuers", emptyNS, "")
	if err != nil {
		log.Panic(err)
	}
	for _, clusterissuer := range clusterIssuers {
		fullPath := fmt.Sprintf("%s/clusterissuer-%s", path, clusterissuer)
		fmt.Println(fullPath)
		aws.UploadFile(bucketName, fullPath, clusterissuer)
	}

	return "Backuped Cert-Manager resources finished successfully!", nil
}
