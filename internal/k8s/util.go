package k8s

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ReturnJobObject returns a matching appsv1.StatefulSet object based on the filters
func ReturnJobObject(clientset *kubernetes.Clientset, namespace string, jobName string) (*batchv1.Job, error) {
	job, err := clientset.BatchV1().Jobs(namespace).Get(context.Background(), jobName, metav1.GetOptions{})
	if err != nil {
		return &batchv1.Job{}, err
	}

	return job, nil
}

// WaitForJobComplete waits for a target Job to reach completion
func WaitForJobComplete(clientset *kubernetes.Clientset, job *batchv1.Job, timeoutSeconds int64) (bool, error) {
	// Format list for metav1.ListOptions for watch
	watchOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf(
			"metadata.name=%s", job.Name),
	}

	// Create watch operation
	objWatch, err := clientset.
		BatchV1().
		Jobs(job.ObjectMeta.Namespace).
		Watch(context.Background(), watchOptions)
	if err != nil {
		log.Fatal().Msgf("error when attempting to wait for Job: %s", err)
	}
	log.Info().Msgf("waiting for %s Job completion. This could take up to %v seconds.", job.Name, timeoutSeconds)

	// Feed events using provided channel
	objChan := objWatch.ResultChan()

	// Listen until the Job is complete
	// Timeout if it isn't complete within timeoutSeconds
	for {
		select {
		case event, ok := <-objChan:
			if !ok {
				// Error if the channel closes
				log.Error().Msgf("failed to wait for job %s to complete", job.Name)
			}
			if event.
				Object.(*batchv1.Job).
				Status.Succeeded > 0 {
				log.Info().Msgf("job %s completed at %s.", job.Name, event.Object.(*batchv1.Job).Status.CompletionTime)
				return true, nil
			}
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			log.Error().Msg("the operation timed out while waiting for the Job to complete")
			return false, errors.New("the operation timed out while waiting for the Job to complete")
		}
	}
}
