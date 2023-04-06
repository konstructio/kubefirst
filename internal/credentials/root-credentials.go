/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package credentials

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/kubefirst/kubefirst/internal/helpers"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
)

// EvalAuth determines whether or not there are active kubefirst platforms
// If there are not, an error is returned
func EvalAuth(expectedCloudProvider string, expectedGitProvider string) (bool, error) {
	flags := helpers.GetCompletionFlags()

	if !flags.SetupComplete {
		return false, fmt.Errorf(
			"There are no active kubefirst platforms to retrieve credentials for.\n\tTo get started, run: kubefirst %s create -h\n",
			expectedCloudProvider,
		)
	}

	switch {
	case flags.CloudProvider == "" || flags.GitProvider == "":
		return false, fmt.Errorf("could not parse cloud and git provider information from config")
	case flags.CloudProvider != expectedCloudProvider:
		return false, fmt.Errorf("it looks like the current deployed platform is %s - try running this command for that provider", flags.CloudProvider)
	}

	log.Info().Msgf("Verified %s platform using %s - parsing credentials...", expectedCloudProvider, expectedGitProvider)

	return true, nil
}

// ParseAuthData gets base root credentials for platform components
func ParseAuthData(clientset *kubernetes.Clientset, cloudProvider string, gitProvider string) error {
	// Retrieve vault root token
	var vaultRootToken string
	vaultUnsealSecretData, err := k8s.ReadSecretV2(clientset, "vault", "vault-unseal-secret")
	if err != nil {
		log.Warn().Msgf("vault secret may not exist: %s", err)
	}
	if len(vaultUnsealSecretData) != 0 {
		vaultRootToken = vaultUnsealSecretData["root-token"]
	}

	// Retrieve argocd password
	var argoCDPassword string
	argoCDSecretData, err := k8s.ReadSecretV2(clientset, "argocd", "argocd-initial-admin-secret")
	if err != nil {
		return err
	}
	argoCDPassword = argoCDSecretData["password"]

	// Retrieve kbot password
	kbotPassword := viper.GetString("kbot.password")

	// Format parameters for final output
	params := make(map[string]string)
	paramsSorted := make(map[string]string)

	// Each item from the objects above should be added to params
	params["ArgoCD Admin Password"] = argoCDPassword
	params["KBot User Password"] = kbotPassword

	if vaultRootToken != "" {
		params["Vault Root Token"] = vaultRootToken
	}

	// Sort
	paramKeys := make([]string, 0, len(params))
	for k := range params {
		paramKeys = append(paramKeys, k)
	}
	sort.Strings(paramKeys)
	for _, k := range paramKeys {
		paramsSorted[k] = params[k]
	}

	messageHeader := fmt.Sprintf("%s Authentication\n\nKeep this data secure. These passwords can be used to access the following applications in your platform.", cloudProvider)
	message := printAuthData(messageHeader, params)
	fmt.Println(reports.StyleMessage(message))

	return nil
}

// printAuthData provides visual output detailing authentication data for k3d
func printAuthData(messageHeader string, params map[string]string) string {
	var createAuthData bytes.Buffer
	createAuthData.WriteString(strings.Repeat("-", 70))
	createAuthData.WriteString(fmt.Sprintf("\n%s\n", messageHeader))
	createAuthData.WriteString(strings.Repeat("-", 70))
	createAuthData.WriteString("\n\n")

	for object, auth := range params {
		createAuthData.WriteString(fmt.Sprintf("%s: %s\n\n", object, auth))
	}

	return createAuthData.String()

}
