package argocd

import (
	"context"
	"errors"
	"fmt"

	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/rs/zerolog/log"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	argoCDInstallPath string = "github.com:argoproj/argo-cd.git/manifests/ha/cluster-install?ref=v2.6.4"
)

// ApplyArgoCDKustomize
func ApplyArgoCDKustomize(clientset *kubernetes.Clientset) error {
	enabled := true
	name := "argocd-bootstrap"
	namespace := "argocd"

	nsObj := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	_, err := clientset.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), nsObj, metav1.CreateOptions{})
		if err != nil {
			log.Error().Err(err).Msg("")
			return errors.New("error creating namespace")
		}
		log.Info().Msgf("namespace created: %s", namespace)
	} else {
		log.Warn().Msgf("namespace %s already exists - skipping", namespace)
	}

	// Create ServiceAccount
	serviceAccount, err := clientset.CoreV1().ServiceAccounts(namespace).Create(context.Background(), &v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		AutomountServiceAccountToken: &enabled,
	}, metav1.CreateOptions{})
	if err != nil {
		log.Error().Msgf("error creating service account: %s", err)
		return err
	}
	log.Info().Msg("created argocd bootstrap service account")

	// Create ClusterRole
	_, err = clientset.RbacV1().ClusterRoles().Create(context.Background(), &rbacv1.ClusterRole{
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
	}, metav1.CreateOptions{})
	if err != nil {
		log.Error().Msgf("error creating role: %s", err)
		return err
	}
	log.Info().Msg("created argocd bootstrap role")

	// Create ClusterRoleBinding
	_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.Background(), &rbacv1.ClusterRoleBinding{
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
	}, metav1.CreateOptions{})
	if err != nil {
		log.Error().Msgf("error creating role binding: %s", err)
		return err
	}
	log.Info().Msg("created argocd bootstrap role binding")

	// Create Job
	backoffLimit := int32(1)
	job, err := clientset.BatchV1().Jobs(namespace).Create(context.Background(), &batchv1.Job{
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
	}, metav1.CreateOptions{})
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
		log.Error().Msgf("could not clean up argocd bootstrap service account %s - manual removal is required", serviceAccount.Name)
	}
	err = clientset.RbacV1().ClusterRoles().Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		log.Error().Msgf("could not clean up argocd bootstrap cluster role %s - manual removal is required", serviceAccount.Name)
	}
	err = clientset.RbacV1().ClusterRoleBindings().Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		log.Error().Msgf("could not clean up argocd bootstrap cluster role binding %s - manual removal is required", serviceAccount.Name)
	}

	return nil
}
