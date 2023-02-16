package vault

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kubefirst/kubefirst/internal/k8s"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	httpClient http.Client
)

const (
	// Rate at which to check for Vault health
	checkInterval int = 10
	// Vault API endpoints
	vaultHealthEndpoint string = "/v1/sys/health"
	vaultInitEndpoint   string = "/v1/sys/init"
	vaultRaftEndpoint   string = "/sys/storage/raft"
	vaultUnsealEndpoint string = "/v1/sys/unseal"
	// For raft, vault-0 will always be primary referenced by its endpoint
	vaultRaftPrimaryAddress string = "http://vault-0.vault-internal"
	// Name for the Secret that gets created that contains root auth data
	vaultSecretName string = "vault-unseal-secret"
	// Namespace that Vault runs in
	vaultNamespace string = "vault"
)

// UnsealVault attempts to initialize and unseal a Vault server
func UnsealVault(kubeConfigPath string, o *VaultUnsealOptions) {
	switch {
	case o.HighAvailability && o.HighAvailabilityType == "raft":
		switch {
		case o.RaftLeader:
			// The leader will be vault-0 and will be conffigured via port-forward to the Service
			runUnseal(kubeConfigPath, *o)
		case o.RaftFollower:
			// Followers will be configured via cli
			initResponse, err := fetchVaultExistingSecretData(kubeConfigPath)
			if err != nil {
				panic(err.Error())
			}

			// Join Vault nodes to raft cluster and unseal
			for i := 1; i < o.Nodes; i++ {
				// Join nodes to cluster
				log.Printf("Joining vault-%d to raft cluster...", i)
				podSessionOpts := k8s.PodSessionOptions{
					Command:    []string{"/bin/sh", "-c", fmt.Sprintf("vault operator raft join %s:8200", vaultRaftPrimaryAddress)},
					Namespace:  vaultNamespace,
					PodName:    fmt.Sprintf("vault-%d", i),
					TtyEnabled: true,
				}
				err = k8s.PodExecSession(kubeConfigPath, &podSessionOpts)
				if err != nil {
					log.Printf("Error running command on Vault Pod: %s", err)
				}
				fmt.Println()

				// Unseal
				for keyNum, rk := range initResponse.Keys {
					if keyNum < 3 {
						log.Printf("Passing key %d...", keyNum+1)
						podSessionOpts := k8s.PodSessionOptions{
							Command:    []string{"/bin/sh", "-c", fmt.Sprintf("vault operator unseal %s", rk)},
							Namespace:  vaultNamespace,
							PodName:    fmt.Sprintf("vault-%d", i),
							TtyEnabled: true,
						}
						err = k8s.PodExecSession(kubeConfigPath, &podSessionOpts)
						if err != nil {
							log.Printf("Error running command on Vault Pod: %s", err)
						}
						fmt.Println()
					} else {
						break
					}

				}
			}
			log.Print("All Vault Pods initialized and unsealed.")
		}
	case o.HighAvailability && o.HighAvailabilityType != "raft":
		log.Printf("Unsupported high-availability setting: %s", o.HighAvailabilityType)
		os.Exit(1)
	}
}

// fetchVaultExistingSecretData looks for an existing vault-unseal Secret and returns its
// data if found
func fetchVaultExistingSecretData(kubeConfigPath string) (InitResponse, error) {
	existingKubernetesSecret, err := k8s.ReadSecretV2(kubeConfigPath, vaultSecretName)
	if err != nil {
		log.Printf("Error reading existing Secret data: %s", err)
		return InitResponse{}, nil
	}

	// Add root-unseal-key entries to slice
	var rkSlice []string
	for key, value := range existingKubernetesSecret {
		if strings.Contains(key, "root-unseal-key-") {
			rkSlice = append(rkSlice, value)
		}
	}

	// Build InitResponse for unseal operation
	initResponse := InitResponse{
		Keys:      rkSlice,
		RootToken: existingKubernetesSecret["root-token"],
	}

	return initResponse, nil
}

// runUnseal carries out the initial unseal action
func runUnseal(kubeConfigPath string, o VaultUnsealOptions) {
	log.Println("Attempting to initialize and unseal Vault instance...")

	switch o.UseAPI {
	case true:
		checkIntervalDuration := time.Duration(checkInterval) * time.Second

		httpClient = http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

		for {
			select {
			// Exit if instructed to by os
			case <-sigChan:
				log.Printf("Shutting down")
				os.Exit(0)
			default:
			}

			// Make request to Vault health endpoint
			response, err := httpClient.Get(o.VaultAPIAddress + vaultHealthEndpoint)

			// Wait until a healthy response is received
			if err != nil {
				log.Printf("Error connecting to Vault: %s", err)
				log.Printf("Next check in %s", checkIntervalDuration)
				time.Sleep(checkIntervalDuration)
				continue
			}
			defer response.Body.Close()

			// Parse health response body
			healthRequestResponseBody, err := ioutil.ReadAll(response.Body)
			if err != nil {
				panic(err)
			}
			var healthResponse HealthResponse
			err = json.Unmarshal(healthRequestResponseBody, &healthResponse)
			if err != nil {
				panic(err)
			}

			// Switch based on health response
			switch {
			case healthResponse.Initialized && !healthResponse.Sealed:
				log.Println("Vault is initialized and unsealed.")
				os.Exit(0)
			case !healthResponse.Sealed && healthResponse.Standby:
				log.Println("Vault is unsealed and in standby mode. Waiting for non-standby transition...")
			case !healthResponse.Initialized && healthResponse.Sealed:
				log.Println("Vault is not initialized and sealed. Initializing and unsealing...")
				initResponse, err := vaultInit(kubeConfigPath, &o)
				if err != nil {
					log.Printf("Unable to init or unseal vault: %s", err)
					os.Exit(1)
				}

				// Unseal
				vaultUnseal(&o, initResponse)
			case healthResponse.Initialized && healthResponse.Sealed:
				log.Println("Vault is initialized but sealed. Unsealing...")
				// Fetch existing Secret value since that's the only reason in this context
				// that Vault would be initialized but not unsealed
				// This is mostly a failsafe for now
				initResponse, err := fetchVaultExistingSecretData(kubeConfigPath)
				if err != nil {
					panic(err.Error())
				}

				// Unseal
				vaultUnseal(&o, initResponse)
			default:
				log.Printf("Vault is in an unknown state. Status code: %d", response.StatusCode)
			}

			select {
			// Exit if instructed to by os
			case <-sigChan:
				log.Printf("Shutting down")
				os.Exit(0)
			// Retry if nothing above worked
			case <-time.After(checkIntervalDuration):
			}
		}
	case false:

	}
}

// vaultInit attempts to initialize a Vault server
func vaultInit(kubeConfigPath string, o *VaultUnsealOptions) (InitResponse, error) {
	// Build InitRequest to be sent to Vault API
	initRequest := InitRequest{
		SecretShares:    5,
		SecretThreshold: 3,
	}

	initRequestData, err := json.Marshal(&initRequest)
	if err != nil {
		log.Println(err)
		return InitResponse{}, err
	}

	r := bytes.NewReader(initRequestData)
	request, err := http.NewRequest("PUT", o.VaultAPIAddress+vaultInitEndpoint, r)
	if err != nil {
		log.Println(err)
		return InitResponse{}, err
	}

	// Submit init request to Vailt API
	response, err := httpClient.Do(request)
	if err != nil {
		log.Println(err)
		return InitResponse{}, err
	}
	defer response.Body.Close()

	// Parse response
	initRequestResponseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return InitResponse{}, err
	}

	if response.StatusCode != 200 {
		log.Printf(
			"Encountered non %d status code during Vault init: %s",
			response.StatusCode,
			string(initRequestResponseBody),
		)
		return InitResponse{}, err
	}

	var initResponse InitResponse
	if err := json.Unmarshal(initRequestResponseBody, &initResponse); err != nil {
		log.Println(err)
		return InitResponse{}, err
	}

	dataToWrite := make(map[string][]byte)
	dataToWrite["root-token"] = []byte(initResponse.RootToken)
	for i, value := range initResponse.Keys {
		dataToWrite[fmt.Sprintf("root-unseal-key-%v", i+1)] = []byte(value)
	}
	secret := v1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      vaultSecretName,
			Namespace: vaultNamespace,
		},
		Data: dataToWrite,
	}

	err = k8s.CreateSecretV2(kubeConfigPath, &secret)
	if err != nil {
		panic(err)
	}

	log.Println("Initialization complete.")

	return initResponse, err
}

// vaultUnseal attempts to unseal a Vault server
func vaultUnseal(o *VaultUnsealOptions, initResponse InitResponse) error {
	for i, key := range initResponse.Keys {
		log.Printf("Providing key %d to Vault API for unseal...", i+1)
		done, err := vaulUnsealTransaction(o, key)
		if done {
			return err
		}
		if err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

// vaultUnsealTransaction provides a single key toward satisfying minimum reqs
func vaulUnsealTransaction(o *VaultUnsealOptions, key string) (bool, error) {
	// Build UnsealRequest to be sent to Vault API
	unsealRequest := UnsealRequest{
		Key: key,
	}

	unsealRequestData, err := json.Marshal(&unsealRequest)
	if err != nil {
		return false, err
	}

	r := bytes.NewReader(unsealRequestData)
	request, err := http.NewRequest(http.MethodPut, o.VaultAPIAddress+vaultUnsealEndpoint, r)
	if err != nil {
		return false, err
	}

	// Submit unseal request to Vault API
	response, err := httpClient.Do(request)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()

	// Parse response
	unsealRequestResponseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	var unsealResponse UnsealResponse
	err = json.Unmarshal(unsealRequestResponseBody, &unsealResponse)
	if err != nil {
		panic(err)
	}

	// When unsealed, indicate to user and exit
	if !unsealResponse.Sealed {
		log.Print("Vault has been unsealed.")
		secretWarning := fmt.Sprintf(`
WARNING: The root token and root unseal keys have been 
written to a Kubernetes Secret called %s - please copy these to a secure 
location outside of the cluster and delete this Secret once cluster setup
is complete.`, vaultSecretName)
		log.Print(secretWarning)
		return true, nil
	}

	return false, nil
}
