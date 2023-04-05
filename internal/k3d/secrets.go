package k3d

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kubefirst/kubefirst/internal/k8s"
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
	// var containerRegistryHost string
	// switch gitProvider {
	// case "github":
	// 	containerRegistryHost = "https://ghcr.io/"
	// case "gitlab":
	// 	containerRegistryHost = "registry.gitlab.io"
	// }

	// todo audit
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

	// Create secrets
	createSecrets := []*v1.Secret{
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
