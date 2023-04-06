/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package argocd

import (
	"context"
	"fmt"
	"time"

	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/rs/zerolog/log"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ApplyArgoCDKustomize
func ApplyArgoCDKustomize(clientset *kubernetes.Clientset, argoCDInstallPath string) error {
	enabled := true
	name := "argocd-bootstrap"
	namespace := "argocd"

	// Create Namespace
	nsObj := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	_, err := clientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		_, err = clientset.CoreV1().Namespaces().Create(context.Background(), nsObj, metav1.CreateOptions{})
		if err != nil {
			log.Error().Msgf("error creating namespace: %s", err)
			return fmt.Errorf("error creating namespace: %s", err)
		}
		log.Info().Msgf("namespace created: %s", namespace)
	} else {
		log.Warn().Msgf("namespace %s already exists - skipping", namespace)
	}

	// Create ServiceAccount
	saObj := &v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		AutomountServiceAccountToken: &enabled,
	}
	_, err = clientset.CoreV1().ServiceAccounts(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		_, err = clientset.CoreV1().ServiceAccounts(namespace).Create(context.Background(), saObj, metav1.CreateOptions{})
		if err != nil {
			log.Error().Msgf("error creating service account: %s", err)
			return fmt.Errorf("error creating service account: %s", err)
		}
		log.Info().Msgf("service account created: %s", name)
	} else {
		log.Warn().Msgf("service account %s already exists - skipping", name)
	}

	// Create ClusterRole
	crObj := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"*"},
				APIGroups: []string{"*"},
				Resources: []string{"*"},
			},
		},
	}
	_, err = clientset.RbacV1().ClusterRoles().Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		_, err = clientset.RbacV1().ClusterRoles().Create(context.Background(), crObj, metav1.CreateOptions{})
		if err != nil {
			log.Error().Msgf("error creating cluster role: %s", err)
			return fmt.Errorf("error creating cluster role: %s", err)
		}
		log.Info().Msgf("cluster role created: %s", name)
	} else {
		log.Warn().Msgf("cluster role %s already exists - skipping", name)
	}

	// Create ClusterRoleBinding
	crbObj := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      name,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     name,
		},
	}
	_, err = clientset.RbacV1().ClusterRoleBindings().Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.Background(), crbObj, metav1.CreateOptions{})
		if err != nil {
			log.Error().Msgf("error creating cluster role binding: %s", err)
			return fmt.Errorf("error creating cluster role binding: %s", err)
		}
		log.Info().Msgf("cluster role binding created: %s", name)
	} else {
		log.Warn().Msgf("cluster role binding %s already exists - skipping", name)
	}

	// Create Job
	backoffLimit := int32(1)
	jobObj := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kustomize-apply-argocd",
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "main",
							Image: "bitnami/kubectl",
							Command: []string{
								"/bin/sh",
								"-c",
								fmt.Sprintf("kubectl apply -k '%s'", argoCDInstallPath),
							},
						},
					},
					ServiceAccountName: name,
					RestartPolicy:      "Never",
				},
			},
			BackoffLimit: &backoffLimit,
		},
	}
	existingJob, err := clientset.BatchV1().Jobs(namespace).Get(context.Background(), jobObj.Name, metav1.GetOptions{})
	if err == nil {
		// Delete the job if it already exists because it likely failed on its last run
		log.Info().Msgf("deleting job %s since it already exists - it will be recreated", jobObj.Name)
		clientset.BatchV1().Jobs(namespace).Delete(context.Background(), existingJob.Name, metav1.DeleteOptions{})
		time.Sleep(time.Second * 5)
	}
	job, err := clientset.BatchV1().Jobs(namespace).Create(context.Background(), jobObj, metav1.CreateOptions{})
	if err != nil {
		log.Error().Msgf("error creating job: %s", err)
		return err
	}
	log.Info().Msg("created argocd bootstrap job")

	// Wait for the Job to finish
	_, err = k8s.WaitForJobComplete(clientset, job, 240)
	if err != nil {
		log.Fatal().Msgf("could not run argocd bootstrap job: %s", err)
	}

	// Cleanup
	err = clientset.CoreV1().ServiceAccounts(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		log.Error().Msgf("could not clean up argocd bootstrap service account %s - manual removal is required", saObj.Name)
	}
	err = clientset.RbacV1().ClusterRoles().Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		log.Error().Msgf("could not clean up argocd bootstrap cluster role %s - manual removal is required", crObj.Name)
	}
	err = clientset.RbacV1().ClusterRoleBindings().Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		log.Error().Msgf("could not clean up argocd bootstrap cluster role binding %s - manual removal is required", crbObj.Name)
	}

	return nil
}
