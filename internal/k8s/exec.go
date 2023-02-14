package k8s

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReturnStatefulSetObject returns a matching v1.StatefulSet object based on the filters
func ReturnStatefulSetObject(kubeConfigPath string, instanceOf string, namespace string, timeoutSeconds float64) (*appsv1.StatefulSet, error) {
	clientset, err := GetClientSet(false, kubeConfigPath)
	if err != nil {
		return nil, err
	}

	// Filter
	statefulSetListOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", instanceOf),
	}

	// Create watch operation
	objWatch, err := clientset.
		AppsV1().
		StatefulSets(namespace).
		Watch(context.Background(), statefulSetListOptions)
	if err != nil {
		log.Fatal().Msgf("Error when attempting to search for StatefulSet: %s", err)
	}
	log.Info().Msgf("Waiting for %s StatefulSet to be created.", instanceOf)

	objChan := objWatch.ResultChan()
	for {
		select {
		case event, ok := <-objChan:
			time.Sleep(time.Second * 1)
			if !ok {
				// Error if the channel closes
				log.Fatal().Msgf("Error waiting for StatefulSet %s to be created: %s", instanceOf, err)
			}
			if event.
				Object.(*appsv1.StatefulSet).Status.Replicas > 0 {
				spec, err := clientset.AppsV1().StatefulSets(namespace).List(context.Background(), statefulSetListOptions)
				if err != nil {
					log.Fatal().Msgf("Error when looking for StatefulSet: %s", err)
					return nil, err
				}
				return &spec.Items[0], nil
			}
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			log.Error().Msg("The StatefulSet was not created within the timeout period.")
			return nil, errors.New("The StatefulSet was not created within the timeout period.")
		}
	}
}

// WaitForStatefulSetReady waits for a target StatefulSet to become ready
func WaitForStatefulSetReady(kubeConfigPath string, statefulset *appsv1.StatefulSet, timeoutSeconds int64) (bool, error) {
	clientset, err := GetClientSet(false, kubeConfigPath)
	if err != nil {
		return false, err
	}

	// Format list for metav1.ListOptions for watch
	configuredReplicas := statefulset.Status.Replicas
	watchOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf(
			"app.kubernetes.io/instance=%s", statefulset.ObjectMeta.Name),
	}

	// Create watch operation
	objWatch, err := clientset.
		AppsV1().
		StatefulSets(statefulset.ObjectMeta.Namespace).
		Watch(context.Background(), watchOptions)
	if err != nil {
		log.Fatal().Msgf("Error when attempting to wait for StatefulSet: %s", err)
	}
	log.Info().Msgf("Waiting for StatefulSet %s to be ready. This could take up to %v seconds.", statefulset.Name, timeoutSeconds)

	objChan := objWatch.ResultChan()
	for {
		select {
		case event, ok := <-objChan:
			time.Sleep(time.Second * 1)
			if !ok {
				// Error if the channel closes
				log.Fatal().Msgf("Error waiting StatefulSet: %s", err)
			}
			if event.
				Object.(*appsv1.StatefulSet).
				Status.AvailableReplicas == configuredReplicas {
				log.Info().Msgf("All Pods in StatefulSet %s are ready.", statefulset.Name)
				return true, nil
			}
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			log.Error().Msg("The StatefulSet was not ready within the timeout period.")
			return false, errors.New("The StatefulSet was not ready within the timeout period.")
		}
	}
}
