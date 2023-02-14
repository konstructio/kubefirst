package k8s

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReturnPodObject returns a matching v1.Pod object based on the filters
func ReturnPodObject(kubeConfigPath string, matchLabel string, matchLabelValue string, namespace string, timeoutSeconds float64) (*v1.Pod, error) {
	clientset, err := GetClientSet(false, kubeConfigPath)
	if err != nil {
		return nil, err
	}

	// Filter
	podListOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", matchLabel, matchLabelValue),
	}

	// Create watch operation
	objWatch, err := clientset.
		CoreV1().
		Pods(namespace).
		Watch(context.Background(), podListOptions)
	if err != nil {
		log.Fatal().Msgf("Error when attempting to search for Pod: %s", err)
	}
	log.Info().Msgf("Waiting for %s Pod to be created.", matchLabelValue)

	objChan := objWatch.ResultChan()
	for {
		select {
		case event, ok := <-objChan:
			time.Sleep(time.Second * 1)
			if !ok {
				// Error if the channel closes
				log.Fatal().Msgf("Error waiting for %s Pod to be created: %s", matchLabelValue, err)
			}
			if event.
				Object.(*v1.Pod).Status.Phase == "Pending" {
				spec, err := clientset.CoreV1().Pods(namespace).List(context.Background(), podListOptions)
				if err != nil {
					log.Fatal().Msgf("Error when searching for Pod: %s", err)
					return nil, err
				}
				return &spec.Items[0], nil
			}
			if event.
				Object.(*v1.Pod).Status.Phase == "Running" {
				spec, err := clientset.CoreV1().Pods(namespace).List(context.Background(), podListOptions)
				if err != nil {
					log.Fatal().Msgf("Error when searching for Pod: %s", err)
					return nil, err
				}
				return &spec.Items[0], nil
			}
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			log.Error().Msg("The Pod was not created within the timeout period.")
			return nil, errors.New("The Pod was not created within the timeout period.")
		}
	}
}

// ReturnStatefulSetObject returns a matching appsv1.StatefulSet object based on the filters
func ReturnStatefulSetObject(kubeConfigPath string, matchLabel string, matchLabelValue string, namespace string, timeoutSeconds float64) (*appsv1.StatefulSet, error) {
	clientset, err := GetClientSet(false, kubeConfigPath)
	if err != nil {
		return nil, err
	}

	// Filter
	statefulSetListOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", matchLabel, matchLabelValue),
	}

	// Create watch operation
	objWatch, err := clientset.
		AppsV1().
		StatefulSets(namespace).
		Watch(context.Background(), statefulSetListOptions)
	if err != nil {
		log.Fatal().Msgf("Error when attempting to search for StatefulSet: %s", err)
	}
	log.Info().Msgf("Waiting for %s StatefulSet to be created.", matchLabelValue)

	objChan := objWatch.ResultChan()
	for {
		select {
		case event, ok := <-objChan:
			time.Sleep(time.Second * 1)
			if !ok {
				// Error if the channel closes
				log.Fatal().Msgf("Error waiting for %s StatefulSet to be created: %s", matchLabelValue, err)
			}
			if event.
				Object.(*appsv1.StatefulSet).Status.Replicas > 0 {
				spec, err := clientset.AppsV1().StatefulSets(namespace).List(context.Background(), statefulSetListOptions)
				if err != nil {
					log.Fatal().Msgf("Error when searching for StatefulSet: %s", err)
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

// WaitForPodReady waits for a target Pod to become ready
func WaitForPodReady(kubeConfigPath string, pod *v1.Pod, timeoutSeconds int64) (bool, error) {
	clientset, err := GetClientSet(false, kubeConfigPath)
	if err != nil {
		return false, err
	}

	// Format list for metav1.ListOptions for watch
	watchOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf(
			"metadata.name=%s", pod.Name),
	}

	// Create watch operation
	objWatch, err := clientset.
		CoreV1().
		Pods(pod.ObjectMeta.Namespace).
		Watch(context.Background(), watchOptions)
	if err != nil {
		log.Fatal().Msgf("Error when attempting to wait for Pod: %s", err)
	}
	log.Info().Msgf("Waiting for %s Pod to be ready. This could take up to %v seconds.", pod.Name, timeoutSeconds)

	// Feed events using provided channel
	objChan := objWatch.ResultChan()

	// Listen until the Pod is ready
	// Timeout if it isn't ready within timeoutSeconds
	for {
		select {
		case event, ok := <-objChan:
			if !ok {
				// Error if the channel closes
				log.Error().Msg("fail")
			}
			if event.
				Object.(*v1.Pod).
				Status.
				Phase == "Running" {
				log.Info().Msgf("Pod %s is %s.", pod.Name, event.Object.(*v1.Pod).Status.Phase)
				return true, nil
			}
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			log.Error().Msg("The operation timed out while waiting for the Pod to become ready.")
			return false, errors.New("The operation timed out while waiting for the Pod to become ready.")
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
		FieldSelector: fmt.Sprintf(
			"metadata.name=%s", statefulset.Name),
	}

	// Create watch operation
	objWatch, err := clientset.
		AppsV1().
		StatefulSets(statefulset.ObjectMeta.Namespace).
		Watch(context.Background(), watchOptions)
	if err != nil {
		log.Fatal().Msgf("Error when attempting to wait for StatefulSet: %s", err)
	}
	log.Info().Msgf("Waiting for %s StatefulSet to be ready. This could take up to %v seconds.", statefulset.Name, timeoutSeconds)

	objChan := objWatch.ResultChan()
	for {
		select {
		case event, ok := <-objChan:
			time.Sleep(time.Second * 1)
			if !ok {
				// Error if the channel closes
				log.Fatal().Msgf("Error waiting for StatefulSet: %s", err)
			}
			if event.
				Object.(*appsv1.StatefulSet).
				Status.ReadyReplicas == configuredReplicas {
				log.Info().Msgf("All Pods in StatefulSet %s are ready.", statefulset.Name)
				return true, nil
			}
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			log.Error().Msg("The StatefulSet was not ready within the timeout period.")
			return false, errors.New("The StatefulSet was not ready within the timeout period.")
		}
	}
}
