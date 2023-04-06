/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package argocd

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// RestPatchArgoCD uses RESTClient to patch a given ArgoCD Application using the provided payload
func RestPatchArgoCD(clientset kubernetes.Interface, applicationName string, payload []byte) error {
	// Call the API to patch the resource
	_, err := clientset.CoreV1().RESTClient().Patch(types.JSONPatchType).
		AbsPath(fmt.Sprintf("/apis/%s", ArgoCDAPIVersion)).
		Namespace("argocd").
		Resource("applications").
		Name(applicationName).
		Body(payload).
		DoRaw(context.Background())
	if err != nil {
		return err
	}
	log.Info().Msgf("patched %s successfully", applicationName)

	return nil
}
