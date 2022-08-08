package k8s

import (
	"log"

	"github.com/kubefirst/kubefirst/configs"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// GetClientSet - get client se for k8s client
func GetClientSet() (*kubernetes.Clientset, error) {
	config := configs.ReadConfig()

	kubeconfig, err := clientcmd.BuildConfigFromFlags("", config.KubeConfigPath)
	if err != nil {
		log.Printf("Error getting kubeconfig: %s", err)
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		log.Printf("Error getting clientset: %s", err)
		return clientset, err
	}

	return clientset, nil
}
