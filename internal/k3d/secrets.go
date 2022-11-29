package k3d

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"os"

	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AddK3DSecrets(dryrun bool) error {
	clientset, err := k8s.GetClientSet(dryrun)

	newNamespaces := []string{"argo", "argocd", "atlantis", "chartmuseum", "github-runner", "vault", "development", "staging", "production"}
	for i, s := range newNamespaces {
		namespace := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: s}}
		_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
		if err != nil {
			log.Error().Err(err).Msg("")
			return errors.New("error creating namespace")
		}
		log.Info().Msgf("%d, %s", i, s)
		log.Info().Msgf("Namespace Created: %s", s)
	}

	dataArgo := map[string][]byte{
		"accesskey": []byte("k-ray"),
		"secretkey": []byte("feedkraystars"),
	}
	argoSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "minio-creds", Namespace: "argo"},
		Data:       dataArgo,
	}
	_, err = clientset.CoreV1().Secrets("argo").Create(context.TODO(), argoSecret, metav1.CreateOptions{})
	if err != nil {
		log.Error().Err(err).Msg("")
		return errors.New("error creating kubernetes secret: argo/minio-creds")
	}
	viper.Set("kubernetes.argo-minio.secret.created", true)
	viper.WriteConfig()

	dataArgoCiSecrets := map[string][]byte{
		"BASIC_AUTH_USER":       []byte("k-ray"),
		"BASIC_AUTH_PASS":       []byte("feedkraystars"),
		"USERNAME":              []byte(viper.GetString("github.user")),
		"PERSONAL_ACCESS_TOKEN": []byte(os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")),
		"username":              []byte(viper.GetString("github.user")),
		"password":              []byte(os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")),
	}
	argoCiSecrets := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "ci-secrets", Namespace: "argo"},
		Data:       dataArgoCiSecrets,
	}
	_, err = clientset.CoreV1().Secrets("argo").Create(context.TODO(), argoCiSecrets, metav1.CreateOptions{})
	if err != nil {
		log.Error().Err(err).Msg("")
		return errors.New("error creating kubernetes secret: argo/ci-secrets")
	}
	viper.Set("kubernetes.argo-ci.secret.created", true)
	viper.WriteConfig()

	usernamePasswordString := fmt.Sprintf("%s:%s", viper.GetString("github.user"), os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN"))
	usernamePasswordStringB64 := base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))

	dockerConfigString := fmt.Sprintf(`{"auths": {"https://ghcr.io/": {"auth": "%s"}}}`, usernamePasswordStringB64)
	argoDockerSecrets := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "docker-config", Namespace: "argo"},
		Data:       map[string][]byte{"config.json": []byte(dockerConfigString)},
	}
	_, err = clientset.CoreV1().Secrets("argo").Create(context.TODO(), argoDockerSecrets, metav1.CreateOptions{})
	if err != nil {
		log.Error().Err(err).Msg("")
		return errors.New("error creating kubernetes secret: argo/docker-config")
	}

	developmentDockerSecrets := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "docker-config", Namespace: "development"},
		Type:       "kubernetes.io/dockerconfigjson",
		Data:       map[string][]byte{".dockerconfigjson": []byte(dockerConfigString)},
	}
	_, err = clientset.CoreV1().Secrets("development").Create(context.TODO(), developmentDockerSecrets, metav1.CreateOptions{})
	if err != nil {
		log.Error().Err(err).Msg("")
		return errors.New("error creating kubernetes secret: development/docker-config")
	}

	stagingDockerSecrets := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "docker-config", Namespace: "staging"},
		Type:       "kubernetes.io/dockerconfigjson",
		Data:       map[string][]byte{".dockerconfigjson": []byte(dockerConfigString)},
	}
	_, err = clientset.CoreV1().Secrets("staging").Create(context.TODO(), stagingDockerSecrets, metav1.CreateOptions{})
	if err != nil {
		log.Error().Err(err).Msg("")
		return errors.New("error creating kubernetes secret: staging/docker-config")
	}

	productionDockerSecrets := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "docker-config", Namespace: "production"},
		Type:       "kubernetes.io/dockerconfigjson",
		Data:       map[string][]byte{".dockerconfigjson": []byte(dockerConfigString)},
	}
	_, err = clientset.CoreV1().Secrets("production").Create(context.TODO(), productionDockerSecrets, metav1.CreateOptions{})
	if err != nil {
		log.Error().Err(err).Msg("")
		return errors.New("error creating kubernetes secret: production/docker-config")
	}
	viper.Set("kubernetes.argo-docker.secret.created", true)
	viper.WriteConfig()

	dataArgoCd := map[string][]byte{
		"password": []byte(os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")),
		"url":      []byte(fmt.Sprintf("https://%s/%s/gitops.git", viper.GetString("github.host"), viper.GetString("github.owner"))),
		"username": []byte(viper.GetString("github.user")),
	}

	argoCdSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "github-repo-creds",
			Namespace:   "argocd",
			Annotations: map[string]string{"managed-by": "argocd.argoproj.io"},
			Labels:      map[string]string{"argocd.argoproj.io/secret-type": "repo-creds"},
		},
		Data: dataArgoCd,
	}
	_, err = clientset.CoreV1().Secrets("argocd").Create(context.TODO(), argoCdSecret, metav1.CreateOptions{})
	if err != nil {
		log.Error().Err(err).Msg("")
		return errors.New("error creating kubernetes secret: argo/minio-creds")
	}
	viper.Set("kubernetes.argo-minio.secret.created", true)
	viper.WriteConfig()

	dataAtlantis := map[string][]byte{
		"ATLANTIS_GH_TOKEN":                   []byte(os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")),
		"ATLANTIS_GH_USER":                    []byte(viper.GetString("github.user")),
		"ATLANTIS_GH_HOSTNAME":                []byte(viper.GetString("github.host")),
		"ATLANTIS_GH_WEBHOOK_SECRET":          []byte(viper.GetString("github.atlantis.webhook.secret")),
		"ARGOCD_AUTH_USERNAME":                []byte("admin"),
		"ARGOCD_INSECURE":                     []byte("true"),
		"ARGOCD_SERVER":                       []byte("http://localhost:8080"),
		"ARGO_SERVER_URL":                     []byte("argo.argo.svc.cluster.local:443"),
		"GITHUB_OWNER":                        []byte(viper.GetString("github.owner")),
		"GITHUB_TOKEN":                        []byte(os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")),
		"TF_VAR_atlantis_repo_webhook_secret": []byte(viper.GetString("github.atlantis.webhook.secret")),
		"TF_VAR_atlantis_repo_webhook_url":    []byte(viper.GetString("github.atlantis.webhook.url")),
		"TF_VAR_email_address":                []byte(viper.GetString("adminemail")),
		"TF_VAR_github_token":                 []byte(os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")),
		"TF_VAR_kubefirst_bot_ssh_public_key": []byte(viper.GetString("botpublickey")),
		"TF_VAR_vault_addr":                   []byte("http://vault.vault.svc.cluster.local:8200"),
		"TF_VAR_vault_token":                  []byte("k1_local_vault_token"),
		"VAULT_ADDR":                          []byte("http://vault.vault.svc.cluster.local:8200"),
		"VAULT_TOKEN":                         []byte("k1_local_vault_token"),
	}
	atlantisSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "atlantis-secrets", Namespace: "atlantis"},
		Data:       dataAtlantis,
	}
	_, err = clientset.CoreV1().Secrets("atlantis").Create(context.TODO(), atlantisSecret, metav1.CreateOptions{})
	if err != nil {
		log.Error().Err(err).Msg("")
		return errors.New("error creating kubernetes secret: atlantis/atlantis-secrets")
	}
	viper.Set("kubernetes.atlantis.secret.created", true)
	viper.WriteConfig()

	dataChartmuseum := map[string][]byte{
		"BASIC_AUTH_USER":       []byte("k-ray"),
		"BASIC_AUTH_PASS":       []byte("feedkraystars"),
		"AWS_ACCESS_KEY_ID":     []byte("k-ray"),
		"AWS_SECRET_ACCESS_KEY": []byte("feedkraystars"),
	}
	chartmuseumSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "chartmuseum", Namespace: "chartmuseum"},
		Data:       dataChartmuseum,
	}
	_, err = clientset.CoreV1().Secrets("chartmuseum").Create(context.TODO(), chartmuseumSecret, metav1.CreateOptions{})
	if err != nil {
		log.Error().Err(err).Msg("")
		return errors.New("error creating kubernetes secret: chartmuseum/chartmuseum")
	}
	viper.Set("kubernetes.chartmuseum.secret.created", true)
	viper.WriteConfig()

	dataGh := map[string][]byte{
		"github_token": []byte(os.Getenv("KUBEFIRST_GITHUB_AUTH_TOKEN")),
	}
	ghRunnerSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "controller-manager", Namespace: "github-runner"},
		Data:       dataGh,
	}
	_, err = clientset.CoreV1().Secrets("github-runner").Create(context.TODO(), ghRunnerSecret, metav1.CreateOptions{})
	if err != nil {
		log.Error().Err(err).Msg("")
		return errors.New("error creating kubernetes secret: github-runner/controller-manager")
	}
	viper.Set("kubernetes.github-runner.secret.created", true)
	viper.WriteConfig()

	vaultData := map[string][]byte{
		"token": []byte("k1_local_vault_token"),
	}
	vaultSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "vault-token", Namespace: "vault"},
		Data:       vaultData,
	}
	_, err = clientset.CoreV1().Secrets("vault").Create(context.TODO(), vaultSecret, metav1.CreateOptions{})
	if err != nil {
		log.Error().Err(err).Msg("")
		return errors.New("error creating kubernetes secret: github-runner/controller-manager")
	}
	viper.Set("kubernetes.vault.secret.created", true)
	viper.WriteConfig()

	return nil
}
