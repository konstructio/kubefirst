package k8s

import (
	"errors"
	"fmt"
)

// VerifyArgoCDReadiness waits for critical resources within ArgoCD to be ready
// and only returns once they're all healthy
//
// This helps prevent race conditions and timeouts
func VerifyArgoCDReadiness(kubeconfigPath string, highAvailabilityEnabled bool) (bool, error) {
	// Wait for ArgoCD StatefulSet Pods to transition to Running
	argoCDStatefulSet, err := ReturnStatefulSetObject(
		kubeconfigPath,
		"app.kubernetes.io/part-of",
		"argocd",
		"argocd",
		120,
	)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Error finding ArgoCD StatefulSet: %s", err))
	}
	_, err = WaitForStatefulSetReady(kubeconfigPath, argoCDStatefulSet, 120, false)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Error waiting for ArgoCD StatefulSet ready state: %s", err))
	}

	// Wait for additional ArgoCD Pods to transition to Running
	// This is related to a condition where apps attempt to deploy before
	// repo, redis, or other health checks are passing
	//
	// This can cause future steps to break since the registry app
	// may never apply

	// argocd-repo-server
	argoCDRepoDeployment, err := ReturnDeploymentObject(
		kubeconfigPath,
		"app.kubernetes.io/name",
		"argocd-repo-server",
		"argocd",
		120,
	)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Error finding ArgoCD repo deployment: %s", err))
	}
	_, err = WaitForDeploymentReady(kubeconfigPath, argoCDRepoDeployment, 120)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Error waiting for ArgoCD repo deployment ready state: %s", err))
	}

	if highAvailabilityEnabled {
		// argocd-redis-ha-haproxy Deployment
		argoCDRedisHAhaproxyDeployment, err := ReturnDeploymentObject(
			kubeconfigPath,
			"app.kubernetes.io/name",
			"argocd-redis-ha-haproxy",
			"argocd",
			120,
		)
		if err != nil {
			return false, errors.New(fmt.Sprintf("Error finding ArgoCD argocd-redis-ha-haproxy deployment: %s", err))
		}
		_, err = WaitForDeploymentReady(kubeconfigPath, argoCDRedisHAhaproxyDeployment, 120)
		if err != nil {
			return false, errors.New(fmt.Sprintf("Error waiting for ArgoCD argocd-redis-ha-haproxy deployment ready state: %s", err))
		}

		// argocd-redis-ha StatefulSet
		argoCDRedisHAServerStatefulSet, err := ReturnStatefulSetObject(
			kubeconfigPath,
			"app.kubernetes.io/name",
			"argocd-redis-ha",
			"argocd",
			120,
		)
		if err != nil {
			return false, errors.New(fmt.Sprintf("Error finding ArgoCD argocd-redis-ha StatefulSet: %s", err))
		}
		_, err = WaitForStatefulSetReady(kubeconfigPath, argoCDRedisHAServerStatefulSet, 120, false)
		if err != nil {
			return false, errors.New(fmt.Sprintf("Error waiting for ArgoCD argocd-redis-ha StatefulSet ready state: %s", err))
		}
	}

	return true, nil
}
