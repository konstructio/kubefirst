package k3d

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AddK3DSecrets(dryrun bool) error {
	clientset, err := k8s.GetClientSet(dryrun)

	newNamespaces := []string{"github-runner", "atlantis"}
	for i, s := range newNamespaces {
		namespace := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: s}}
		_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
		if err != nil {
			log.Println("Error:", s)
			return errors.New("error creating namespace")
		}
		log.Println(i, s)
		log.Println("Namespace Created:", s)
	}

	dataGh := map[string][]byte{
		"github_token": []byte(os.Getenv("GITHUB_AUTH_TOKEN")),
	}
	ghRunnerSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "controller-manager", Namespace: "github-runner"},
		Data:       dataGh,
	}
	_, err = clientset.CoreV1().Secrets("github-runner").Create(context.TODO(), ghRunnerSecret, metav1.CreateOptions{})
	if err != nil {
		log.Println("Error:", err)
		return errors.New("error creating kubernetes secret: github-runner/controller-manager")
	}
	viper.Set("kubernetes.github-runner.secret.created", true)
	viper.WriteConfig()

	dataAtlantis := map[string][]byte{
		"ATLANTIS_GH_TOKEN":          []byte(os.Getenv("GITHUB_AUTH_TOKEN")),
		"ATLANTIS_GH_USER":           []byte(viper.GetString("github.user")),
		"ATLANTIS_GH_HOSTNAME":       []byte(viper.GetString("github.host")),
		"ATLANTIS_GH_WEBHOOK_SECRET": []byte(viper.GetString("github.atlantis.webhook.secret")),
	}
	ghAtlantisSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "atlantis-secrets", Namespace: "atlantis"},
		Data:       dataAtlantis,
	}
	_, err = clientset.CoreV1().Secrets("atlantis").Create(context.TODO(), ghAtlantisSecret, metav1.CreateOptions{})
	if err != nil {
		log.Println("Error:", err)
		return errors.New("error creating kubernetes secret: atlantis/atlantis-secrets")
	}
	viper.Set("kubernetes.atlantis-secrets.secret.created", true)
	viper.WriteConfig()
	return nil
}
