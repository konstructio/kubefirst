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
		// todo: this is hardcoded / testing
		"ATLANTIS_ATLANTIS_URL": []byte(viper.GetString("http://localhost:4141")),

		// todo: testing / clean up
		"GITHUB_OWNER":                        []byte(viper.GetString("github.org")),
		"GITHUB_TOKEN":                        []byte(os.Getenv("GITHUB_AUTH_TOKEN")),
		"TF_VAR_atlantis_repo_webhook_secret": []byte(viper.GetString("github.atlantis.webhook.secret")),
		"TF_VAR_email_address":                []byte(viper.GetString("adminemail")),
		"TF_VAR_github_token":                 []byte(os.Getenv("GITHUB_AUTH_TOKEN")),
		"TF_VAR_kubefirst_bot_ssh_public_key": []byte(viper.GetString("botpublickey")),
		"TF_VAR_ssh_private_key":              []byte(viper.GetString("botprivatekey")),
		"TF_VAR_vault_addr":                   []byte(viper.GetString("vault.local.service")),
		"TF_VAR_vault_token":                  []byte("k1_local_vault_token"),
		"VAULT_ADDR":                          []byte(viper.GetString("vault.local.service")),
		"VAULT_TOKEN":                         []byte("k1_local_vault_token"),
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
