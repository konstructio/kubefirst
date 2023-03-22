package k3d

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AddK3DSecrets(
	atlantisWebhookSecret string,
	kbotPublicKey string,
	destinationGitopsRepoGitURL string,
	kbotPrivateKey string,
	dryRun bool,
	gitProvider string,
	gitUser string,
	gitOwner string,
	kubeconfigPath string,
	tokenValue string,
) error {
	clientset, err := k8s.GetClientSet(dryRun, kubeconfigPath)
	if err != nil {
		log.Info().Msg("error getting kubernetes clientset")
	}

	// Set git provider token value
	var containerRegistryHost string
	switch gitProvider {
	case "github":
		containerRegistryHost = "https://ghcr.io/"
	case "gitlab":
		containerRegistryHost = "registry.gitlab.io"
	}

	newNamespaces := []string{
		"atlantis",
		"external-dns",
		"external-secrets-operator",
		fmt.Sprintf("%s-runner", gitProvider),
	}

	for i, s := range newNamespaces {
		namespace := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: s}}
		_, err := clientset.CoreV1().Namespaces().Get(context.TODO(), s, metav1.GetOptions{})
		if err != nil {
			_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
			if err != nil {
				log.Error().Err(err).Msg("")
				return errors.New("error creating namespace")
			}
			log.Info().Msgf("%d, %s", i, s)
			log.Info().Msgf("namespace created: %s", s)
		} else {
			log.Warn().Msgf("namespace %s already exists - skipping", s)
		}
	}

	// Data used for secret creation
	// docker auth
	usernamePasswordString := fmt.Sprintf("%s:%s", gitUser, tokenValue)
	usernamePasswordStringB64 := base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))
	dockerConfigString := fmt.Sprintf(`{"auths": {"%s": {"auth": "%s"}}}`, containerRegistryHost, usernamePasswordStringB64)

	// Create secrets
	createSecrets := []*v1.Secret{
		// argo
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ci-secrets", Namespace: "argo"},
			Data: map[string][]byte{
				"BASIC_AUTH_USER":       []byte(pkg.MinioDefaultUsername),
				"BASIC_AUTH_PASS":       []byte(pkg.MinioDefaultPassword),
				"USERNAME":              []byte(gitUser),
				"PERSONAL_ACCESS_TOKEN": []byte(tokenValue),
				"username":              []byte(gitUser),
				"password":              []byte(tokenValue),
			},
		},
		// argocd
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "repo-credentials-template",
				Namespace:   "argocd",
				Annotations: map[string]string{"managed-by": "argocd.argoproj.io"},
				Labels:      map[string]string{"argocd.argoproj.io/secret-type": "repository"},
			},
			Data: map[string][]byte{
				"type":          []byte("git"),
				"name":          []byte(fmt.Sprintf("%s-gitops", gitUser)),
				"url":           []byte(destinationGitopsRepoGitURL),
				"sshPrivateKey": []byte(kbotPrivateKey),
			},
		},
		// atlantis
		{
			ObjectMeta: metav1.ObjectMeta{Name: "atlantis-secrets", Namespace: "atlantis"},
			Data: map[string][]byte{
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
				"TF_VAR_email_address":                []byte("your@email.com"),
				"TF_VAR_github_token":                 []byte(tokenValue),
				"TF_VAR_kbot_ssh_public_key":          []byte(kbotPublicKey),
				"TF_VAR_vault_addr":                   []byte("http://vault.vault.svc.cluster.local:8200"),
				"TF_VAR_vault_token":                  []byte("k1_local_vault_token"),
				"VAULT_ADDR":                          []byte("http://vault.vault.svc.cluster.local:8200"),
				"VAULT_TOKEN":                         []byte("k1_local_vault_token"),
			},
		},
		// atlantis ngrok (k3d-ngrok)
		{
			ObjectMeta: metav1.ObjectMeta{Name: "k3d-ngrok", Namespace: "atlantis"},
			Data: map[string][]byte{
				"GIT_PROVIDER": []byte(gitProvider),
				"GIT_OWNER":    []byte(gitOwner),
				"GIT_TOKEN":    []byte(tokenValue),
				// This is the only webhook we need for this step
				"GIT_REPOSITORY": []byte("gitops"),
			},
		},
		// chartmuseum
		{
			ObjectMeta: metav1.ObjectMeta{Name: "chartmuseum-secrets", Namespace: "chartmuseum"},
			Data: map[string][]byte{
				"BASIC_AUTH_USER":       []byte(pkg.MinioDefaultUsername),
				"BASIC_AUTH_PASS":       []byte(pkg.MinioDefaultPassword),
				"AWS_ACCESS_KEY_ID":     []byte(pkg.MinioDefaultUsername),
				"AWS_SECRET_ACCESS_KEY": []byte(pkg.MinioDefaultPassword),
			},
		},
		// git runner
		{
			ObjectMeta: metav1.ObjectMeta{Name: "controller-manager", Namespace: fmt.Sprintf("%s-runner", gitProvider)},
			Data: map[string][]byte{
				fmt.Sprintf("%s_token", gitProvider): []byte(tokenValue),
			},
		},
		// minio
		{
			ObjectMeta: metav1.ObjectMeta{Name: "minio-creds", Namespace: "argo"},
			Data: map[string][]byte{
				"accesskey": []byte(pkg.MinioDefaultUsername),
				"secretkey": []byte(pkg.MinioDefaultPassword),
			},
		},
		// vault
		{
			ObjectMeta: metav1.ObjectMeta{Name: "vault-token", Namespace: "vault"},
			Data: map[string][]byte{
				"token": []byte("k1_local_vault_token"),
			},
		},
		// argo docker config
		{
			ObjectMeta: metav1.ObjectMeta{Name: "docker-config", Namespace: "argo"},
			Type:       "kubernetes.io/dockerconfigjson",
			Data:       map[string][]byte{".dockerconfigjson": []byte(dockerConfigString)},
		},
		// development docker config
		{
			ObjectMeta: metav1.ObjectMeta{Name: "docker-config", Namespace: "development"},
			Type:       "kubernetes.io/dockerconfigjson",
			Data:       map[string][]byte{".dockerconfigjson": []byte(dockerConfigString)},
		},
		// production docker config
		{
			ObjectMeta: metav1.ObjectMeta{Name: "docker-config", Namespace: "production"},
			Type:       "kubernetes.io/dockerconfigjson",
			Data:       map[string][]byte{".dockerconfigjson": []byte(dockerConfigString)},
		},
		// staging docker config
		{
			ObjectMeta: metav1.ObjectMeta{Name: "docker-config", Namespace: "staging"},
			Type:       "kubernetes.io/dockerconfigjson",
			Data:       map[string][]byte{".dockerconfigjson": []byte(dockerConfigString)},
		},
	}
	for _, secret := range createSecrets {
		_, err := clientset.CoreV1().Secrets(secret.ObjectMeta.Namespace).Get(context.TODO(), secret.ObjectMeta.Name, metav1.GetOptions{})
		if err == nil {
			log.Info().Msgf("kubernetes secret %s/%s already created - skipping", secret.Namespace, secret.Name)
		} else if strings.Contains(err.Error(), "not found") {
			_, err = clientset.CoreV1().Secrets(secret.ObjectMeta.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
			if err != nil {
				log.Fatal().Msgf("error creating kubernetes secret %s/%s: %s", secret.Namespace, secret.Name, err)
			}
			log.Info().Msgf("created kubernetes secret: %s/%s", secret.Namespace, secret.Name)
		}
	}

	// Data used for service account creation
	var automountServiceAccountToken bool = true

	// Create service accounts
	createServiceAccounts := []*v1.ServiceAccount{
		// atlantis
		{
			ObjectMeta:                   metav1.ObjectMeta{Name: "atlantis", Namespace: "atlantis"},
			AutomountServiceAccountToken: &automountServiceAccountToken,
		},
		// external-secrets-operator
		{
			ObjectMeta: metav1.ObjectMeta{Name: "external-secrets", Namespace: "external-secrets-operator"},
		},
	}
	for _, serviceAccount := range createServiceAccounts {
		_, err := clientset.CoreV1().ServiceAccounts(serviceAccount.ObjectMeta.Namespace).Get(context.TODO(), serviceAccount.ObjectMeta.Name, metav1.GetOptions{})
		if err == nil {
			log.Info().Msgf("kubernetes service account %s/%s already created - skipping", serviceAccount.Namespace, serviceAccount.Name)
		} else if strings.Contains(err.Error(), "not found") {
			_, err = clientset.CoreV1().ServiceAccounts(serviceAccount.ObjectMeta.Namespace).Create(context.TODO(), serviceAccount, metav1.CreateOptions{})
			if err != nil {
				log.Fatal().Msgf("error creating kubernetes service account %s/%s: %s", serviceAccount.Namespace, serviceAccount.Name, err)
			}
			log.Info().Msgf("created kubernetes service account: %s/%s", serviceAccount.Namespace, serviceAccount.Name)
		}
	}

	return nil
}
