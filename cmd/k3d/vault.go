/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/kubefirst/kubefirst/internal/progress"
	"github.com/kubefirst/runtime/pkg/helpers"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
)

const (
	// Name for the Secret that gets created that contains root auth data
	vaultSecretName string = "vault-unseal-secret"
	// Namespace that Vault runs in
	vaultNamespace string = "vault"
	// number of secret threshold Vault unseal
	secretThreshold = 3
)

// unsealVault will attempt to unseal vaule again if it is currently unsealed
func unsealVault(cmd *cobra.Command, args []string) error {
	flags := helpers.GetClusterStatusFlags()
	if !flags.SetupComplete {
		return fmt.Errorf("there doesn't appear to be an active k3d cluster")
	}
	config := k3d.GetConfig(
		viper.GetString("flags.cluster-name"),
		flags.GitProvider,
		viper.GetString(fmt.Sprintf("flags.%s-owner", flags.GitProvider)),
		flags.GitProtocol,
	)
	kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

	// Vault api client
	vaultClient, err := vaultapi.NewClient(&vaultapi.Config{
		Address: "https://vault.kubefirst.dev",
	})
	if err != nil {
		return err
	}
	vaultClient.CloneConfig().ConfigureTLS(&vaultapi.TLSConfig{
		Insecure: true,
	})

	// Determine vault health
	health, err := vaultClient.Sys().Health()
	if err != nil {
		return err
	}

	switch health.Sealed {
	case true:
		node := "vault-0"
		existingInitResponse, err := parseExistingVaultInitSecret(kcfg.Clientset)
		if err != nil {
			return err
		}

		sealStatusTracking := 0
		for i, shard := range existingInitResponse.Keys {
			if i < secretThreshold {
				log.Info().Msgf("passing unseal shard %v to %s", i+1, node)
				deadline := time.Now().Add(60 * time.Second)
				ctx, cancel := context.WithDeadline(context.Background(), deadline)
				defer cancel()
				// Try 5 times to pass unseal shard
				for i := 0; i < 5; i++ {
					_, err := vaultClient.Sys().UnsealWithContext(ctx, shard)
					if err != nil {
						if errors.Is(err, context.DeadlineExceeded) {
							continue
						}
					}
					if i == 5 {
						return fmt.Errorf("error passing unseal shard %v to %s: %s", i+1, node, err)
					}
				}
				// Wait for key acceptance
				for i := 0; i < 10; i++ {
					sealStatus, err := vaultClient.Sys().SealStatus()
					if err != nil {
						return fmt.Errorf("error retrieving health of %s: %s", node, err)
					}
					if sealStatus.Progress > sealStatusTracking || !sealStatus.Sealed {
						log.Info().Msgf("shard accepted")
						sealStatusTracking += 1
						break
					}
					log.Info().Msgf("waiting for node %s to accept unseal shard", node)
					time.Sleep(time.Second * 6)
				}
			}
		}

		fmt.Printf("vault unsealed\n")
	case false:
		return fmt.Errorf("vault is already unsealed")
	}

	progress.Progress.Quit()

	return nil
}

// parseExistingVaultInitSecret returns the value of a vault initialization secret if it exists
func parseExistingVaultInitSecret(clientset *kubernetes.Clientset) (*vaultapi.InitResponse, error) {
	// If vault has already been initialized, the response is formatted to contain the value
	// of the initialization secret
	secret, err := k8s.ReadSecretV2(clientset, vaultNamespace, vaultSecretName)
	if err != nil {
		return &vaultapi.InitResponse{}, err
	}

	// Add root-unseal-key entries to slice
	var rkSlice []string
	for key, value := range secret {
		if strings.Contains(key, "root-unseal-key-") {
			rkSlice = append(rkSlice, value)
		}
	}

	existingInitResponse := &vaultapi.InitResponse{
		Keys:      rkSlice,
		RootToken: secret["root-token"],
	}
	return existingInitResponse, nil
}
