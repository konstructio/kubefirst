package vault

import (
	"fmt"
	"strings"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (conf *VaultConfiguration) AutoUnseal() (*vaultapi.InitResponse, error) {
	vaultClient, err := vaultapi.NewClient(&conf.Config)
	if err != nil {
		return &vaultapi.InitResponse{}, err
	}
	vaultClient.CloneConfig().ConfigureTLS(&vaultapi.TLSConfig{
		Insecure: true,
	})
	log.Info().Msg("created vault client, initializing vault with auto unseal")

	initResponse, err := vaultClient.Sys().Init(&vaultapi.InitRequest{
		RecoveryShares:    RecoveryShares,
		RecoveryThreshold: RecoveryThreshold,
		SecretShares:      SecretShares,
		SecretThreshold:   SecretThreshold,
	})
	log.Info().Msg("vault initialization complete")

	return initResponse, err
}

// UnsealRaftLeader initializes and unseals a vault leader when using raft for ha and storage
func (conf *VaultConfiguration) UnsealRaftLeader(clientset *kubernetes.Clientset, kubeConfigPath string) error {
	//* vault port-forward
	log.Info().Msgf("starting port-forward for vault-0")
	vaultStopChannel := make(chan struct{}, 1)
	defer func() {
		close(vaultStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		kubeConfigPath,
		"vault-0",
		"vault",
		8200,
		8200,
		vaultStopChannel,
	)
	time.Sleep(time.Second * 2)

	// Vault api client
	vaultClient, err := vaultapi.NewClient(&conf.Config)
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

	switch health.Initialized {
	case false:
		log.Info().Msg("initializing vault raft leader")

		initResponse, err := vaultClient.Sys().Init(&vaultapi.InitRequest{
			RecoveryShares:    RecoveryShares,
			RecoveryThreshold: RecoveryThreshold,
			SecretShares:      SecretShares,
			SecretThreshold:   SecretThreshold,
		})
		if err != nil {
			return err
		}

		// Write secret containing init data
		dataToWrite := make(map[string][]byte)
		dataToWrite["root-token"] = []byte(initResponse.RootToken)
		for i, value := range initResponse.Keys {
			dataToWrite[fmt.Sprintf("root-unseal-key-%v", i+1)] = []byte(value)
		}
		secret := v1.Secret{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      VaultSecretName,
				Namespace: VaultNamespace,
			},
			Data: dataToWrite,
		}

		log.Info().Msgf("creating secret %s containing vault initialization data", VaultSecretName)
		err = k8s.CreateSecretV2(clientset, &secret)
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Second * 3)

		// Unseal raft leader
		for i, shard := range initResponse.Keys {
			if i < 3 {
				log.Info().Msgf("passing unseal shard %v to %s", i+1, "vault-0")
				_, err := vaultClient.Sys().Unseal(shard)
				if err != nil {
					return err
				}
			} else {
				break
			}
		}
	case true:
		log.Info().Msgf("%s is already initialized", "vault-0")

		// Determine vault health
		health, err = vaultClient.Sys().Health()
		if err != nil {
			return err
		}

		switch health.Sealed {
		case true:
			existingInitResponse, err := parseExistingVaultInitSecret(kubeConfigPath)
			if err != nil {
				return err
			}

			// Unseal raft leader
			for i, shard := range existingInitResponse.Keys {
				if i < 3 {
					retries := 10
					for r := 0; r < retries; r++ {
						if r > 0 {
							log.Warn().Msgf("encountered an error during unseal, retrying (%d/%d)", r+1, retries)
						}
						time.Sleep(5 * time.Second)

						log.Info().Msgf("passing unseal shard %v to %s", i+1, "vault-0")
						_, err := vaultClient.Sys().Unseal(shard)
						if err != nil {
							continue
						} else {
							break
						}
					}
					time.Sleep(time.Second * 2)
				} else {
					break
				}

			}
		case false:
			log.Info().Msgf("%s is already unsealed", "vault-0")
		}
	}

	log.Info().Msgf("closing port-forward for vault-0")
	time.Sleep(time.Second * 3)

	return nil
}

// UnsealRaftFollowers initializes, unseals, and joins raft followers when using raft for ha and storage
func (conf *VaultConfiguration) UnsealRaftFollowers(kubeConfigPath string) error {
	// With the current iteration of the Vault helm chart, we create 3 nodes
	// vault-0 is unsealed as leader, vault-1 and vault-2 are unsealed here
	raftNodes := []string{"vault-1", "vault-2"}
	existingInitResponse, err := parseExistingVaultInitSecret(kubeConfigPath)
	if err != nil {
		return err
	}

	for _, node := range raftNodes {
		//* vault port-forward
		log.Info().Msgf("starting port-forward for %s", node)
		vaultStopChannel := make(chan struct{}, 1)
		k8s.OpenPortForwardPodWrapper(
			kubeConfigPath,
			node,
			"vault",
			8200,
			8200,
			vaultStopChannel,
		)
		time.Sleep(time.Second * 2)

		// Instantiate vault client per node
		vaultClient, err := vaultapi.NewClient(&conf.Config)
		if err != nil {
			return err
		}
		vaultClient.CloneConfig().ConfigureTLS(&vaultapi.TLSConfig{
			Insecure: true,
		})
		log.Info().Msgf("created vault client for %s", node)

		// Determine vault health
		health, err := vaultClient.Sys().Health()
		if err != nil {
			return err
		}

		switch health.Initialized {
		case false:
			// Join to raft cluster
			log.Info().Msgf("joining raft follower %s to vault cluster", node)
			_, err = vaultClient.Sys().RaftJoin(&vaultapi.RaftJoinRequest{
				//AutoJoin:         "",
				//AutoJoinScheme:   "",
				//AutoJoinPort:     0,
				LeaderAPIAddr: fmt.Sprintf("%s:8200", vaultRaftPrimaryAddress),
				// LeaderCACert:     "",
				// LeaderClientCert: "",
				// LeaderClientKey:  "",
				Retry: true,
			})
			if err != nil {
				return err
			}
			time.Sleep(time.Second * 1)
		case true:
			log.Info().Msgf("raft follower %s is already initialized", node)
		}

		// Determine vault health
		health, err = vaultClient.Sys().Health()
		if err != nil {
			return err
		}
		// Allow time between operations
		time.Sleep(time.Second * 5)

		switch health.Sealed {
		case true:
			// Unseal
			// Unseal raft leader
			for i, shard := range existingInitResponse.Keys {
				if i < 3 {
					retries := 10
					for r := 0; r < retries; r++ {
						if r > 0 {
							log.Warn().Msgf("encountered an error during unseal, retrying (%d/%d)", r+1, retries)
						}
						time.Sleep(5 * time.Second)

						log.Info().Msgf("passing unseal shard %v to %s", i+1, node)
						_, err := vaultClient.Sys().Unseal(shard)
						if err != nil {
							continue
						} else {
							break
						}
					}
					time.Sleep(time.Second * 2)
				} else {
					break
				}
			}
		case false:
			log.Info().Msgf("raft follower %s is already unsealed", node)
		}

		log.Info().Msgf("closing port-forward for %s", node)
		close(vaultStopChannel)

		// Allow time between operations
		time.Sleep(time.Second * 5)
	}

	return nil
}

// parseExistingVaultInitSecret returns the value of a vault initialization secret if it exists
func parseExistingVaultInitSecret(kubeConfigPath string) (*vaultapi.InitResponse, error) {
	// If vault has already been initialized, the response is formatted to contain the value
	// of the initialization secret
	secret, err := k8s.ReadSecretV2(kubeConfigPath, VaultNamespace, VaultSecretName)
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
