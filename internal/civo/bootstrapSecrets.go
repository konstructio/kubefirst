package civo

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func BootstrapCivoMgmtCluster(dryRun bool, kubeconfigPath string) error {
	clientset, err := k8s.GetClientSet(dryRun, kubeconfigPath)
	if err != nil {
		log.Info().Msg("error getting kubernetes clientset")
	}

	newNamespaces := []string{"argo", "argocd", "atlantis", "chartmuseum", "external-dns", "external-secrets-operator", "github-runner", "vault", "development", "staging", "production"}
	for i, s := range newNamespaces {
		namespace := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: s}}
		_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
		if err != nil {
			log.Error().Err(err).Msg("")
			return errors.New("error creating namespace")
		}
		log.Info().Msgf("%d, %s", i, s)
		log.Info().Msgf("namespace created: %s", s)
	}

	dataCivoCreds := map[string][]byte{
		"civo-token": []byte(os.Getenv("CIVO_TOKEN")),
	}
	civoSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "civo-creds", Namespace: "external-dns"},
		Data:       dataCivoCreds,
	}
	_, err = clientset.CoreV1().Secrets("external-dns").Create(context.TODO(), civoSecret, metav1.CreateOptions{})
	if err != nil {
		log.Info().Msgf("Error: %s", err)
		return errors.New("error creating kubernetes secret: external-dns/civo-creds")
	}

	// vault access
	// todo: this won't be necessary after the current fixes are implemented
	// dataExternalSecretsOperator := map[string][]byte{
	// 	"vault-token": []byte("k1_local_vault_token"),
	// }
	// externalSecretOperatorSecret := &v1.Secret{
	// 	ObjectMeta: metav1.ObjectMeta{Name: "vault-token", Namespace: "external-secrets-operator"},
	// 	Data:       dataExternalSecretsOperator,
	// }
	// _, err = clientset.CoreV1().Secrets("external-secrets-operator").Create(context.TODO(), externalSecretOperatorSecret, metav1.CreateOptions{})
	// if err != nil {
	// 	log.Info().Msgf("Error: %s", err)
	// 	return errors.New("error creating kubernetes secret: external-secrets-operator/vault-token")
	// }
	// log.Info().Msg("created secret: external-secrets-operator/vault-token")

	// This is necessary since it's required to be present when configuring the Vault auth
	// for Kubernetes so that permissions can be associated with the ServiceAccount
	externalSecretOperatorServiceAccount := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "external-secrets", Namespace: "external-secrets-operator"},
	}
	_, err = clientset.CoreV1().ServiceAccounts("external-secrets-operator").Create(context.TODO(), externalSecretOperatorServiceAccount, metav1.CreateOptions{})
	if err != nil {
		log.Info().Msgf("Error: %s", err)
		return errors.New("error creating kubernetes service account: external-secrets-operator/external-secrets")
	}
	log.Info().Msg("created service account: external-secrets-operator/external-secrets")

	minioCreds := map[string][]byte{
		"accesskey": []byte("k-ray"),
		"secretkey": []byte("feedkraystars"),
	}
	minioSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "minio-creds", Namespace: "argo"},
		Data:       minioCreds,
	}
	_, err = clientset.CoreV1().Secrets("argo").Create(context.TODO(), minioSecret, metav1.CreateOptions{})
	if err != nil {
		log.Error().Err(err).Msg("")
		return errors.New("error creating kubernetes secret: argo/minio-creds")
	}

	dataArgoCiSecrets := map[string][]byte{
		"accesskey":       []byte(viper.GetString("kubefirst.state-store-creds.access-key-id")),
		"secretkey":       []byte(viper.GetString("kubefirst.state-store-creds.secret-access-key-id")),
		"BASIC_AUTH_USER": []byte("k-ray"),
		"BASIC_AUTH_PASS": []byte("feedkraystars"),
		"SSH_PRIVATE_KEY": []byte(viper.GetString("kbot.private-key")),
	}

	//*
	argoCiSecrets := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "ci-secrets", Namespace: "argo"},
		Data:       dataArgoCiSecrets,
	}
	_, err = clientset.CoreV1().Secrets("argo").Create(context.TODO(), argoCiSecrets, metav1.CreateOptions{})
	if err != nil {
		log.Error().Err(err).Msg("")
		return errors.New("error creating kubernetes secret: argo/ci-secrets")
	}

	usernamePasswordString := fmt.Sprintf("%s:%s", viper.GetString("github.user"), os.Getenv("GITHUB_TOKEN"))
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

	dataArgoCd := map[string][]byte{
		"type":          []byte("git"),
		"name":          []byte(fmt.Sprintf("%s-gitops", viper.GetString("github.owner"))),
		"url":           []byte(viper.GetString("github.repos.gitops.git-url")),
		"sshPrivateKey": []byte(viper.GetString("kbot.private-key")),
	}

	argoCdSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "repo-credentials-template",
			Namespace:   "argocd",
			Annotations: map[string]string{"managed-by": "argocd.argoproj.io"},
			Labels:      map[string]string{"argocd.argoproj.io/secret-type": "repository"},
		},
		Data: dataArgoCd,
	}
	_, err = clientset.CoreV1().Secrets("argocd").Create(context.TODO(), argoCdSecret, metav1.CreateOptions{})
	if err != nil {
		log.Error().Err(err).Msg("")
		return errors.New("error creating kubernetes secret: argo/minio-creds")
	}

	dataAtlantis := map[string][]byte{
		"ATLANTIS_GH_TOKEN":                   []byte(os.Getenv("GITHUB_TOKEN")),
		"ATLANTIS_GH_USER":                    []byte(viper.GetString("github.user")),
		"ATLANTIS_GH_HOSTNAME":                []byte(viper.GetString("github.host")),
		"ATLANTIS_GH_WEBHOOK_SECRET":          []byte(viper.GetString("github.atlantis.webhook.secret")),
		"ARGOCD_AUTH_USERNAME":                []byte("admin"),
		"ARGOCD_INSECURE":                     []byte("true"),
		"ARGOCD_SERVER":                       []byte("http://localhost:8080"),
		"ARGO_SERVER_URL":                     []byte("argo.argo.svc.cluster.local:443"),
		"GITHUB_OWNER":                        []byte(viper.GetString("github.owner")),
		"GITHUB_TOKEN":                        []byte(os.Getenv("GITHUB_TOKEN")),
		"TF_VAR_atlantis_repo_webhook_secret": []byte(viper.GetString("github.atlantis.webhook.secret")),
		"TF_VAR_atlantis_repo_webhook_url":    []byte(viper.GetString("github.atlantis.webhook.url")),
		"TF_VAR_email_address":                []byte(viper.GetString("adminemail")),
		"TF_VAR_github_token":                 []byte(os.Getenv("GITHUB_TOKEN")),
		"TF_VAR_kubefirst_bot_ssh_public_key": []byte(viper.GetString("kbot.public-key")),
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

	dataChartmuseum := map[string][]byte{
		"BASIC_AUTH_USER":       []byte("k-ray"),
		"BASIC_AUTH_PASS":       []byte("feedkraystars"),
		"AWS_ACCESS_KEY_ID":     []byte(viper.GetString("kubefirst.state-store-creds.access-key-id")),
		"AWS_SECRET_ACCESS_KEY": []byte(viper.GetString("kubefirst.state-store-creds.secret-access-key-id")),
	}
	chartmuseumSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "chartmuseum-secrets", Namespace: "chartmuseum"},
		Data:       dataChartmuseum,
	}
	_, err = clientset.CoreV1().Secrets("chartmuseum").Create(context.TODO(), chartmuseumSecret, metav1.CreateOptions{})
	if err != nil {
		log.Error().Err(err).Msg("")
		return errors.New("error creating kubernetes secret: chartmuseum/chartmuseum")
	}

	dataGh := map[string][]byte{
		"github_token": []byte(os.Getenv("GITHUB_TOKEN")),
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
	return nil
}
