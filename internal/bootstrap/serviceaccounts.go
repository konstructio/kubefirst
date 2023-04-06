/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package bootstrap

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ServiceAccounts(clientset *kubernetes.Clientset) error {
	var automountServiceAccountToken bool = true

	// Create namespace
	// Skip if it already exists
	newNamespaces := []string{
		"argo",
		"atlantis",
		"external-secrets-operator",
	}
	for i, s := range newNamespaces {
		namespace := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: s}}
		_, err := clientset.CoreV1().Namespaces().Get(context.TODO(), s, metav1.GetOptions{})
		if err != nil {
			_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
			if err != nil {
				log.Error().Err(err).Msg("")
				return fmt.Errorf("error creating namespace")
			}
			log.Info().Msgf("%d, %s", i, s)
			log.Info().Msgf("namespace created: %s", s)
		} else {
			log.Warn().Msgf("namespace %s already exists - skipping", s)
		}
	}

	// Create service accounts
	createServiceAccounts := []*v1.ServiceAccount{
		// atlantis
		{
			ObjectMeta:                   metav1.ObjectMeta{Name: "atlantis", Namespace: "atlantis"},
			AutomountServiceAccountToken: &automountServiceAccountToken,
		},
		// external-secrets-operator
		{
			ObjectMeta:                   metav1.ObjectMeta{Name: "external-secrets", Namespace: "external-secrets-operator"},
			AutomountServiceAccountToken: &automountServiceAccountToken,
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
