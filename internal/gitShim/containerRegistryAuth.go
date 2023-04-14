package gitShim

import (
	"encoding/base64"
	"fmt"

	"github.com/kubefirst/runtime/pkg/gitlab"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/rs/zerolog/log"
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
		dockerConfigString := fmt.Sprintf(`{"auths": {"%s": {"username": "%s", "password": "%s", "email": "%s", "auth": "%s"}}}`,
			obj.ContainerRegistryHost,
			obj.GithubOwner,
			obj.GitToken,
			"k-bot@example.com",
			usernamePasswordStringB64,
		)

		// Create argo workflows pull secret
		argoDeployTokenSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: "argo"},
			Data:       map[string][]byte{"config.json": []byte(dockerConfigString)},
			Type:       "Opaque",
		}
		err := k8s.CreateSecretV2(obj.Clientset, argoDeployTokenSecret)
		if err != nil {
			log.Error().Msgf("error while creating secret for container registry auth: %s", err)
		}

	// GitLab Deploy Tokens
	// Project deploy tokens are generated for each member of createTokensForProjects
	// These deploy tokens are used to authorize against the GitLab container registry
	case "gitlab":
		gitlabClient, err := gitlab.NewGitLabClient(obj.GitToken, obj.GitlabGroupFlag)
		if err != nil {
			return "", err
		}

		// Create argo workflows pull secret
		var p = gitlab.DeployTokenCreateParameters{
			Name:     secretName,
			Username: secretName,
			Scopes:   []string{"read_registry", "write_registry"},
		}
		token, err := gitlabClient.CreateGroupDeployToken(0, &p)
		if err != nil {
			log.Error().Msgf("error while creating secret for container registry auth: %s", err)
		}

		return token, nil
	}

	return "", nil
}
