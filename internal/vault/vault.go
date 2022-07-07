package vault

import (
	"context"
	"encoding/json"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"
	"log"
)

// GetVaultRootToken get `vault-unseal-keys` token on Vault.
func GetVaultRootToken(vaultSecretClient coreV1Types.SecretInterface) (string, error) {
	name := "vault-unseal-keys"
	log.Printf("Reading secret %s\n", name)
	secret, err := vaultSecretClient.Get(context.TODO(), name, metaV1.GetOptions{})
	if err != nil {
		return "", err
	}

	var vaultRootToken string
	var jsonData map[string]interface{}
	for _, value := range secret.Data {
		if err := json.Unmarshal(value, &jsonData); err != nil {
			return "", err
		}
		vaultRootToken = jsonData["root_token"].(string)
	}
	return vaultRootToken, nil
}
