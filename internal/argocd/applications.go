/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package argocd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	v1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	health "github.com/argoproj/gitops-engine/pkg/health"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
)

const (
	applicationDeletionTimeout int = 120
)

// ArgoCDApplicationCleanup removes and waits for specific ArgoCD applications
func ArgoCDApplicationCleanup(clientset kubernetes.Interface, removeApps []string) error {
	// Patch registry app to remove syncPolicy
	removeSyncPolicyPatch, _ := json.Marshal(
		[]PatchStringValue{{
			Op:    "remove",
			Path:  "/spec/syncPolicy",
			Value: "",
		}})
	err := RestPatchArgoCD(clientset, "registry", removeSyncPolicyPatch)
	if err != nil {
		log.Warn().Msgf("could not remove syncPolicy from registry, it may already be disabled")
	}
	log.Info().Msgf("removed syncPolicy from registry application or it was already disabled")

	// Patch dependent applications to remove syncPolicy}
	for _, app := range removeApps {
		log.Info().Msgf("attempting to delete argocd application %s", app)
		err := waitForApplicationDeletion(clientset, app)
		if err != nil {
			log.Error().Msgf("error deleting argocd application %s: %s", err)
		}
	}

	return nil
}

// deleteArgoCDApplicationV2 deletes an ArgoCD application
func deleteArgoCDApplicationV2(clientset kubernetes.Interface, applicationName string, ch chan<- bool) error {
	// Call the API to delete an ArgoCD application
	data, err := clientset.CoreV1().RESTClient().Delete().
		AbsPath(fmt.Sprintf("/apis/%s", ArgoCDAPIVersion)).
		Namespace("argocd").
		Resource("applications").
		Name(applicationName).
		DoRaw(context.Background())
	if err != nil {
		log.Error().Msgf("error deleting argocd application: %s", err)
	}

	// Unmarshal JSON API response to map[string]interface{}
	var resp map[string]interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		log.Error().Msgf("error deleting argocd application: %s", err)
		return err
	}
	log.Info().Msgf("deleting %s: %s", applicationName, strings.ToLower(fmt.Sprintf("%v", resp["status"])))

	for i := 0; i < applicationDeletionTimeout; i++ {
		status, _ := returnArgoCDApplicationStatus(clientset, applicationName)
		switch status {
		case health.HealthStatusUnknown:
			ch <- true
			return nil
		case health.HealthStatusMissing:
			ch <- true
			return nil
		case health.HealthStatusProgressing:
			log.Info().Msgf("application %s is progressing", applicationName)
			continue
		case health.HealthStatusDegraded:
			log.Info().Msgf("application %s is progressing", applicationName)
			continue
		}
		time.Sleep(time.Second * 1)
	}

	return nil
}

// returnArgoCDApplicationStatus returns the status details of a given ArgoCD application
func returnArgoCDApplicationStatus(clientset kubernetes.Interface, applicationName string) (health.HealthStatusCode, error) {
	// Call the API to return an ArgoCD application object
	data, err := clientset.CoreV1().RESTClient().Get().
		AbsPath(fmt.Sprintf("/apis/%s", ArgoCDAPIVersion)).
		Namespace("argocd").
		Resource("applications").
		Name(applicationName).
		DoRaw(context.Background())
	if err != nil {
		log.Error().Msgf("error retrieving argocd applications: %s", err)
		return health.HealthStatusUnknown, err
	}

	// Unmarshal JSON API response to map[string]interface{}
	var resp *v1alpha1.Application
	if err := json.Unmarshal(data, &resp); err != nil {
		log.Error().Msgf("error converting argocd application data: %s", err)
		return health.HealthStatusUnknown, err
	}
	status := resp.Status.Health.Status

	return status, nil
}

// waitForApplicationDeletion disables sync and deletes specific applications
// from ArgoCD before continuing
func waitForApplicationDeletion(clientset kubernetes.Interface, applicationName string) error {
	ch := make(chan bool)
	// Patch app to remove sync
	removeSyncPolicyPatch, _ := json.Marshal(
		[]PatchStringValue{{
			Op:    "remove",
			Path:  "/spec/syncPolicy",
			Value: "",
		}})
	err := RestPatchArgoCD(clientset, applicationName, removeSyncPolicyPatch)
	if err != nil {
		log.Info().Msgf("error patching argocd application %s: %s", applicationName, err)
	}
	log.Info().Msgf("removed syncPolicy from argocd application %s or it was not present", applicationName)

	// Delete applications and wait for them to report as deleted
	go deleteArgoCDApplicationV2(clientset, applicationName, ch)
	for {
		select {
		case deleted, ok := <-ch:
			if !ok || deleted {
				fmt.Printf("deleted argocd application %s if it existed\n", applicationName)
				return nil
			}
		case <-time.After(time.Duration(applicationDeletionTimeout) * time.Second):
			return fmt.Errorf("timed out waiting for argocd application %s to delete", applicationName)
		}
	}
}
