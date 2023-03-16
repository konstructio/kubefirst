package helpers

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// EvalAuth determines whether or not there are active kubefirst platforms
// If there are not, an error is returned
func EvalAuth(expectedCloudProvider string, expectedGitProvider string) (bool, error) {
	cloudProvider := viper.GetString("kubefirst.cloud-provider")
	gitProvider := viper.GetString("kubefirst.git-provider")
	setupComplete := viper.GetBool("kubefirst.setup-complete")

	if !setupComplete {
		return false, errors.New(
			fmt.Sprintf(
				"There are no active kubefirst platforms to retrieve authentication data for.\n\tTo get started, run: kubefirst %s create -h\n",
				expectedCloudProvider,
			),
		)
	}

	if cloudProvider == "" || gitProvider == "" {
		return false, errors.New("Could not parse cloud and git provider information from config.")
	}
	log.Info().Msgf("Verified %s platform using %s - parsing authentication data...", expectedCloudProvider, expectedGitProvider)

	return true, nil
}

func ParseAuthData(gitProviderFlag string) error {
	// Determine if there are active installs
	gitProvider := viper.GetString("flags.git-provider")
	_, err := EvalAuth(k3d.CloudProvider, gitProvider)
	if err != nil {
		return err
	}

	gitOwner := viper.GetString(fmt.Sprintf("flags.%s-owner", gitProvider))
	if gitOwner == "" {
		return errors.New("could not parse git-owner from .kubefirst")
	}

	config := k3d.GetConfig(gitProviderFlag, gitOwner)

	// Retrieve vault root token
	var vaultRootToken string
	vaultUnsealSecretData, err := k8s.ReadSecretV2(config.Kubeconfig, "vault", "vault-unseal-secret")
	if err != nil {
		return err
	}
	vaultRootToken = vaultUnsealSecretData["root-token"]

	// Retrieve argocd password
	var argoCDPassword string
	argoCDSecretData, err := k8s.ReadSecretV2(config.Kubeconfig, "argocd", "argocd-initial-admin-secret")
	if err != nil {
		return err
	}
	argoCDPassword = argoCDSecretData["password"]

	// Retrieve kbot password
	var kbotPassword string
	kbotPassword = viper.GetString("kbot.password")

	// Format parameters for final output
	params := make(map[string]string)
	paramsSorted := make(map[string]string)

	// Each item from the objects above should be added to params
	params["ArgoCD"] = argoCDPassword
	params["KBot User"] = kbotPassword
	params["Vault"] = vaultRootToken

	// Sort
	paramKeys := make([]string, 0, len(params))
	for k := range params {
		paramKeys = append(paramKeys, k)
	}
	sort.Strings(paramKeys)
	for _, k := range paramKeys {
		paramsSorted[k] = params[k]
	}

	var messageHeader = fmt.Sprintf(
		"K3d Authentication\n\nKeep this data secure. These passwords can be used to access the following applications in your platform.",
	)

	message := printK3dAuthData(messageHeader, params)
	fmt.Println(reports.StyleMessage(message))

	return nil
}

// printK3dAuthData provides visual output detailing authentication data for k3d
func printK3dAuthData(messageHeader string, params map[string]string) string {
	var createK3dAuthData bytes.Buffer
	createK3dAuthData.WriteString(strings.Repeat("-", 70))
	createK3dAuthData.WriteString(fmt.Sprintf("\n%s\n", messageHeader))
	createK3dAuthData.WriteString(strings.Repeat("-", 70))
	createK3dAuthData.WriteString("\n\n")

	for object, auth := range params {
		createK3dAuthData.WriteString(fmt.Sprintf("%s: %s\n\n", object, auth))
	}

	return createK3dAuthData.String()

}
