/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package k8s

import (
	"context"
	"log"

	apiV1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const sparkServiceAccountName = "spark-sa"
const sparkRoleName = "spark-role"

// AddPermissionsForSparkOperator - Add permission to run spark jobs on cluster mode
func AddPermissionsForSparkOperator(namespace string) error {

	clientset, err := GetClientSet()
	if err != nil {
		log.Printf("Error getting clientset: %s", err)
		return err
	}

	// Create SA per name space
	serviceAccountClient := clientset.CoreV1().ServiceAccounts(namespace)
	sa := &apiV1.ServiceAccount{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      sparkServiceAccountName,
			Namespace: namespace,
		},
	}
	createdSA, err := serviceAccountClient.Create(context.TODO(), sa, metaV1.CreateOptions{})
	if err != nil {
		log.Printf("Error creating  SA: %s - %s", err, createdSA)
		return err
	} else {
		log.Printf("Service Account '%v' Created at namespace '%v'.", sa.ObjectMeta.Name, sa.ObjectMeta.Namespace)
	}

	// Create RoleBinding with all subjects
	roleBindingClient := clientset.RbacV1().RoleBindings(namespace)

	roleBinding := &rbacV1.RoleBinding{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name: sparkRoleName + "-" + namespace,
		},
		RoleRef: rbacV1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     sparkRoleName,
		},
		Subjects: []rbacV1.Subject{
			rbacV1.Subject{
				Kind:      "ServiceAccount",
				Name:      sparkServiceAccountName,
				Namespace: namespace,
			},
		},
	}

	createdRoleBinding, err := roleBindingClient.Create(context.TODO(), roleBinding, metaV1.CreateOptions{})
	if err != nil {
		log.Printf("Error creating  RoleBinding: %s - %s", err, createdRoleBinding)
		return err
	} else {
		log.Printf("RoleRoleBinding '%v' Created at namespace '%v'.", sparkRoleName, namespace)
	}
	return nil
}

// RemovePermissionsForSparkOperator - Remove permission to run spark jobs on cluster mode
func RemovePermissionsForSparkOperator(namespace string) error {
	clientset, err := GetClientSet()
	if err != nil {
		log.Printf("Error getting clientset: %s", err)
		return err
	}
	// Create SA per name space
	serviceAccountClient := clientset.CoreV1().ServiceAccounts(namespace)
	err = serviceAccountClient.Delete(context.TODO(), sparkServiceAccountName, metaV1.DeleteOptions{})
	if err != nil {
		log.Printf("Error creating  SA: %s - %s", err, sparkServiceAccountName)
		return err
	} else {
		log.Printf("Service Account '%v' removed at namespace '%v'.", sparkServiceAccountName, namespace)
	}

	roleBindingClient := clientset.RbacV1().RoleBindings(namespace)

	roleBindingName := sparkRoleName + "-" + namespace
	err = roleBindingClient.Delete(context.TODO(), roleBindingName, metaV1.DeleteOptions{})
	if err != nil {
		log.Printf("Error creating  RoleBinding: %s - %s", err, roleBindingName)
		return err
	} else {
		log.Printf("RoleRoleBinding '%v' Created at namespace '%v'.", roleBindingName, namespace)
	}

	return nil

}

func CreateRoleSparkJob(namespace string) error {
	clientset, err := GetClientSet()
	if err != nil {
		log.Printf("Error getting clientset: %s", err)
		return err
	}
	// Create Role
	roleClient := clientset.RbacV1().Roles(namespace)
	var rules = []rbacV1.PolicyRule{
		rbacV1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"*"},
		},
		rbacV1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"services"},
			Verbs:     []string{"*"},
		},
		rbacV1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"configmaps"},
			Verbs:     []string{"*"},
		},
		rbacV1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"*"},
		},
		rbacV1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumeclaims"},
			Verbs:     []string{"*"},
		},
	}

	role := &rbacV1.Role{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      sparkRoleName,
			Namespace: namespace,
		},
		Rules: rules,
	}

	createdRole, err := roleClient.Create(context.TODO(), role, metaV1.CreateOptions{})
	if err != nil {
		log.Printf("Error creating  Role: %s - %s", err, createdRole)
		return err
	} else {
		log.Printf("Role '%v' Created at namespace '%v'.", role.ObjectMeta.Name, role.ObjectMeta.Namespace)
	}
	return nil
}
