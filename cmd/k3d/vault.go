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

	"github.com/hashicorp/vault/api"
	"github.com/konstructio/kubefirst-api/pkg/k3d"
	"github.com/konstructio/kubefirst-api/pkg/k8s"
	utils "github.com/konstructio/kubefirst-api/pkg/utils"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
)

const (
	// Name for the Secret that gets created that contains root auth data
	vaultSecretName = "vault-unseal-secret"
	// Namespace that Vault runs in
	vaultNamespace = "vault"
	// number of secret threshold Vault unseal
	secretThreshold = 3
)

func unsealVault(_ *cobra.Command, _ []string) error {
	flags := utils.GetClusterStatusFlags()
	if !flags.SetupComplete {
		return fmt.Errorf("failed to unseal vault: there doesn't appear to be an active k3d cluster")
	}
	config, err := k3d.GetConfig(
		viper.GetString("flags.cluster-name"),
		flags.GitProvider,
		viper.GetString(fmt.Sprintf("flags.%s-owner", flags.GitProvider)),
		flags.GitProtocol,
	)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	kcfg, err := k8s.CreateKubeConfig(false, config.Kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubeconfig: %w", err)
	}

	vaultClient, err := api.NewClient(&api.Config{
		Address: "https://vault.kubefirst.dev",
	})
	if err != nil {
		return fmt.Errorf("failed to create vault client: %w", err)
	}
	vaultClient.CloneConfig().ConfigureTLS(&api.TLSConfig{
		Insecure: true,
	})

	health, err := vaultClient.Sys().Health()
	if err != nil {
		return fmt.Errorf("failed to check vault health: %w", err)
	}

	if health.Sealed {
		node := "vault-0"
		existingInitResponse, err := parseExistingVaultInitSecret(kcfg.Clientset)
		if err != nil {
			return fmt.Errorf("failed to parse existing vault init secret: %w", err)
		}

		sealStatusTracking := 0
		for i, shard := range existingInitResponse.Keys {
			if i < secretThreshold {
				log.Info().Msgf("passing unseal shard %d to %q", i+1, node)
				deadline := time.Now().Add(60 * time.Second)
				ctx, cancel := context.WithDeadline(context.Background(), deadline)
				defer cancel()
				for j := 0; j < 5; j++ {
					_, err := vaultClient.Sys().UnsealWithContext(ctx, shard)
					if err != nil {
						if errors.Is(err, context.DeadlineExceeded) {
							continue
						}
						return fmt.Errorf("error passing unseal shard %d to %q: %w", i+1, node, err)
					}
				}
				for j := 0; j < 10; j++ {
					sealStatus, err := vaultClient.Sys().SealStatus()
					if err != nil {
						return fmt.Errorf("error retrieving health of %q: %w", node, err)
					}
					if sealStatus.Progress > sealStatusTracking || !sealStatus.Sealed {
						log.Info().Msg("shard accepted")
						sealStatusTracking++
						break
					}
					log.Info().Msgf("waiting for node %q to accept unseal shard", node)
					time.Sleep(6 * time.Second)
				}
			}
		}

		log.Printf("vault unsealed")
	} else {
		return fmt.Errorf("failed to unseal vault: vault is already unsealed")
	}

	return nil
}

func parseExistingVaultInitSecret(clientset kubernetes.Interface) (*api.InitResponse, error) {
	secret, err := k8s.ReadSecretV2(clientset, vaultNamespace, vaultSecretName)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret: %w", err)
	}

	var rkSlice []string
	for key, value := range secret {
		if strings.Contains(key, "root-unseal-key-") {
			rkSlice = append(rkSlice, value)
		}
	}

	existingInitResponse := &api.InitResponse{
		Keys:      rkSlice,
		RootToken: secret["root-token"],
	}
	return existingInitResponse, nil
}
