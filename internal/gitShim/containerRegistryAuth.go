/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gitShim

import (
	"encoding/base64"
	"fmt"

	"github.com/konstructio/kubefirst-api/pkg/gitlab"
	"github.com/konstructio/kubefirst-api/pkg/k8s"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const secretName = "container-registry-auth"

type ContainerRegistryAuth struct {
	GitProvider           string
	GitUser               string
	GitToken              string
	GitlabGroupFlag       string
	GithubOwner           string
	ContainerRegistryHost string

	Clientset *kubernetes.Clientset
}

// CreateContainerRegistrySecret
func CreateContainerRegistrySecret(obj *ContainerRegistryAuth) (string, error) {
	// Handle secret creation for container registry authentication
	switch obj.GitProvider {
	// GitHub docker auth secret
	// kaniko requires a specific format for Docker auth created as a secret
	// For GitHub, this becomes the provided token (pat)
	case "github":
		usernamePasswordString := fmt.Sprintf("%s:%s", obj.GitUser, obj.GitToken)
		usernamePasswordStringB64 := base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))
		dockerConfigString := fmt.Sprintf(`{"auths": {"%s": {"username": %q, "password": %q, "email": %q, "auth": %q}}}`,
			obj.ContainerRegistryHost,
			obj.GithubOwner,
			obj.GitToken,
			"k-bot@example.com",
			usernamePasswordStringB64,
		)

		argoDeployTokenSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: "argo"},
			Data:       map[string][]byte{"config.json": []byte(dockerConfigString)},
			Type:       "Opaque",
		}
		if err := k8s.CreateSecretV2(obj.Clientset, argoDeployTokenSecret); err != nil {
			return "", fmt.Errorf("error while creating secret for GitHub container registry auth: %w", err)
		}

	case "gitlab":
		gitlabClient, err := gitlab.NewGitLabClient(obj.GitToken, obj.GitlabGroupFlag)
		if err != nil {
			return "", fmt.Errorf("error while creating GitLab client: %w", err)
		}

		p := gitlab.DeployTokenCreateParameters{
			Name:     secretName,
			Username: secretName,
			Scopes:   []string{"read_registry", "write_registry"},
		}
		token, err := gitlabClient.CreateGroupDeployToken(0, &p)
		if err != nil {
			return "", fmt.Errorf("error while creating GitLab group deploy token: %w", err)
		}

		return token, nil
	}

	return "", nil
}
