package k3d

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/internal/k8s"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AddK3DSecrets(
	atlantisWebhookSecret string,
	atlantisWebhookURL string,
	kbotPublicKey string,
	destinationGitopsRepoGitURL string,
	kbotPrivateKey string,
	dryRun bool,
	gitProvider string,
	gitUser string,
	kubeconfigPath string,
) error {
	clientset, err := k8s.GetClientSet(dryRun, kubeconfigPath)
	if err != nil {
		log.Info().Msg("error getting kubernetes clientset")
	}

	// Set git provider token value
	var tokenValue, containerRegistryHost string
	switch gitProvider {
	case "github":
		tokenValue = os.Getenv("GITHUB_TOKEN")
		containerRegistryHost = "https://ghcr.io/"
	case "gitlab":
		tokenValue = os.Getenv("GITLAB_TOKEN")
		containerRegistryHost = "registry.gitlab.io"
	}

	newNamespaces := []string{"argo", "argocd", "atlantis", "chartmuseum", "external-dns", fmt.Sprintf("%s-runner", gitProvider), "vault", "development", "staging", "production"}
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
		"BASIC_AUTH_USER":       []byte("k-ray"),
		"BASIC_AUTH_PASS":       []byte("feedkraystars"),
		"USERNAME":              []byte(gitUser),
		"PERSONAL_ACCESS_TOKEN": []byte(tokenValue),
		"username":              []byte(gitUser),
		"password":              []byte(tokenValue),
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

	usernamePasswordString := fmt.Sprintf("%s:%s", gitUser, tokenValue)
	usernamePasswordStringB64 := base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))

	dockerConfigString := fmt.Sprintf(`{"auths": {"%s": {"auth": "%s"}}}`, containerRegistryHost, usernamePasswordStringB64)
	argoDockerSecrets := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "docker-config", Namespace: "argo"},
		Type:       "kubernetes.io/dockerconfigjson",
		Data:       map[string][]byte{".dockerconfigjson": []byte(dockerConfigString)},
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
		"name":          []byte(fmt.Sprintf("%s-gitops", gitUser)),
		"url":           []byte(destinationGitopsRepoGitURL),
		"sshPrivateKey": []byte(kbotPrivateKey),
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
		"ATLANTIS_GH_TOKEN":                   []byte(tokenValue),
		"ATLANTIS_GH_USER":                    []byte(gitUser),
		"ATLANTIS_GH_HOSTNAME":                []byte(fmt.Sprintf("%s.com", gitProvider)),
		"ATLANTIS_GH_WEBHOOK_SECRET":          []byte(atlantisWebhookSecret),
		"ARGOCD_AUTH_USERNAME":                []byte("admin"),
		"ARGOCD_INSECURE":                     []byte("true"),
		"ARGOCD_SERVER":                       []byte("http://localhost:8080"),
		"ARGO_SERVER_URL":                     []byte("argo.argo.svc.cluster.local:443"),
		"GITHUB_OWNER":                        []byte(gitUser),
		"GITHUB_TOKEN":                        []byte(tokenValue),
		"TF_VAR_atlantis_repo_webhook_secret": []byte(atlantisWebhookSecret),
		"TF_VAR_atlantis_repo_webhook_url":    []byte(atlantisWebhookURL),
		"TF_VAR_email_address":                []byte("your@email.com"),
		"TF_VAR_github_token":                 []byte(tokenValue),
		"TF_VAR_kubefirst_bot_ssh_public_key": []byte(kbotPublicKey),
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
		"AWS_ACCESS_KEY_ID":     []byte("k-ray"),
		"AWS_SECRET_ACCESS_KEY": []byte("feedkraystars"),
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

	dataRunner := map[string][]byte{
		fmt.Sprintf("%s_token", gitProvider): []byte(tokenValue),
	}

	runnerSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "controller-manager", Namespace: fmt.Sprintf("%s-runner", gitProvider)},
		Data:       dataRunner,
	}
	_, err = clientset.CoreV1().Secrets(fmt.Sprintf("%s-runner", gitProvider)).Create(context.TODO(), runnerSecret, metav1.CreateOptions{})
	if err != nil {
		log.Error().Err(err).Msg("")
		return errors.New(fmt.Sprintf("error creating kubernetes secret: %s-runner/controller-manager", gitProvider))
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
