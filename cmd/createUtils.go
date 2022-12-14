package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

// todo: move it to internals/ArgoCD
// deprecated
func setArgocdCreds(dryRun bool) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, setArgocdCreds skipped.")
		viper.Set("argocd.admin.password", "dry-run-not-real-pwd")
		viper.Set("argocd.admin.username", "dry-run-not-admin")
		viper.WriteConfig()
		return
	}
	clientset, err := k8s.GetClientSet(dryRun)
	if err != nil {
		panic(err.Error())
	}
	argocd.ArgocdSecretClient = clientset.CoreV1().Secrets("argocd")

	argocdPassword := k8s.GetSecretValue(argocd.ArgocdSecretClient, "argocd-initial-admin-secret", "password")
	if argocdPassword == "" {
		log.Panic().Msg("Missing argocdPassword")
	}

	viper.Set("argocd.admin.password", argocdPassword)
	viper.Set("argocd.admin.username", "admin")
	viper.WriteConfig()
}

// deprecated
func waitArgoCDToBeReady(dryRun bool) {
	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, waitArgoCDToBeReady skipped.")
		return
	}
	config := configs.ReadConfig()
	x := 50
	for i := 0; i < x; i++ {
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "namespace/argocd")
		if err != nil {
			log.Warn().Err(err).Msg("Waiting argocd to be born")
			time.Sleep(10 * time.Second)
		} else {
			log.Info().Msg("argocd namespace found, continuing")
			time.Sleep(5 * time.Second)
			break
		}
	}
	for i := 0; i < x; i++ {
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "get", "pods", "-l", "app.kubernetes.io/name=argocd-server")
		if err != nil {
			log.Warn().Err(err).Msg("Waiting for argocd pods to create, checking in 10 seconds")
			time.Sleep(10 * time.Second)
		} else {
			log.Info().Msg("argocd pods found, waiting for them to be running")
			viper.Set("argocd.ready", true)
			viper.WriteConfig()
			time.Sleep(15 * time.Second)
			break
		}
	}
}

// deprecated
func waitVaultToBeRunning(dryRun bool) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, waitVaultToBeRunning skipped.")
		return
	}
	token := viper.GetString("vault.token")
	if len(token) > 0 {
		log.Info().Msg("Vault token exists, skipping Vault health checks")
		return
	}

	// waits for Vault Namespace
	x := 50
	for i := 0; i < x; i++ {
		isVaultPodCreated, err := k8s.IsNamespaceCreated("vault")
		if !isVaultPodCreated {
			log.Warn().Err(err).Msg("waiting Vault to be born")
			time.Sleep(10 * time.Second)
			continue
		}
		log.Info().Msg("vault namespace found, continuing")
		time.Sleep(25 * time.Second)
		break
	}

	// waits for Vault Pod
	x = 50
	for i := 0; i < x; i++ {
		clientset, err := k8s.GetClientSet(dryRun)
		if err != nil {
			log.Error().Err(err).Msg("")
		}
		vaultPodInterface := clientset.CoreV1().Pods("vault")
		vaultPodName := k8s.GetPodNameByLabel(vaultPodInterface, "app.kubernetes.io/instance=vault")

		if len(vaultPodName) == 0 {
			log.Warn().Err(err).Msg("waiting Vault Pod to be created...")
			time.Sleep(10 * time.Second)
			continue
		}

		log.Warn().Msg("Vault Pod found, continuing...")
		log.Warn().Msg("waiting Vault Pod to be running...")
		time.Sleep(10 * time.Second)
		break
	}

}

// deprecated
func loopUntilPodIsReady(dryRun bool) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, loopUntilPodIsReady skipped.")
		return
	}
	token := viper.GetString("vault.token")
	if len(token) == 0 {

		totalAttempts := 50
		url := "http://localhost:8200/v1/sys/health"
		for i := 0; i < totalAttempts; i++ {
			log.Info().Msgf("vault is not ready yet, sleeping and checking again, attempt (%d/%d)", i+1, totalAttempts)
			time.Sleep(10 * time.Second)

			req, _ := http.NewRequest("GET", url, nil)

			req.Header.Add("Content-Type", "application/json")

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Error().Err(err).Msg("error with http request Do, vault is not available")
				// todo: temporary code
				log.Info().Msg("trying to open port-forward again...")
				go func() {
					_, err := k8s.PortForward(false, "vault", "svc/vault", "8200:8200")
					if err != nil {
						log.Error().Err(err).Msg("error opening Vault port forward")
					}
				}()
				continue
			}

			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			if err != nil {
				log.Error().Err(err).Msg("vault is available but the body is not what is expected ")
				continue
			}

			var responseJson map[string]interface{}

			if err := json.Unmarshal(body, &responseJson); err != nil {
				log.Error().Err(err).Msg("vault is available but the body is not what is expected")
				continue
			}

			_, ok := responseJson["initialized"]
			if ok {
				log.Printf("vault is initialized and is in the expected state")
				return
			}
			log.Panic().Msg("vault was never initialized")
		}
		viper.Set("vault.status.running", true)
		viper.WriteConfig()
	} else {
		log.Info().Msg("vault token already exists, skipping vault health checks loopUntilPodIsReady")
	}
}

type VaultInitResponse struct {
	Initialized                bool   `json:"initialized"`
	Sealed                     bool   `json:"sealed"`
	Standby                    bool   `json:"standby"`
	PerformanceStandby         bool   `json:"performance_standby"`
	ReplicationPerformanceMode string `json:"replication_performance_mode"`
	ReplicationDrMode          string `json:"replication_dr_mode"`
	ServerTimeUtc              int    `json:"server_time_utc"`
	Version                    string `json:"version"`
}

type VaultUnsealResponse struct {
	UnsealKeysB64         []interface{} `json:"unseal_keys_b64"`
	UnsealKeysHex         []interface{} `json:"unseal_keys_hex"`
	UnsealShares          int           `json:"unseal_shares"`
	UnsealThreshold       int           `json:"unseal_threshold"`
	RecoveryKeysBase64    []string      `json:"recovery_keys_base64"`
	RecoveryKeys          []string      `json:"recovery_keys"`
	RecoveryKeysShares    int           `json:"recovery_keys_shares"`
	RecoveryKeysThreshold int           `json:"recovery_keys_threshold"`
	RootToken             string        `json:"root_token"`
	Keys                  []string      `json:"keys"`
	KeysB64               []string      `json:"keys_base64"`
}

func initializeVaultAndAutoUnseal(dryRun bool) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, initializeVaultAndAutoUnseal skipped.")
		return
	}

	token := viper.GetString("vault.token")
	if len(token) == 0 {

		time.Sleep(time.Second * 10)
		url := "http://127.0.0.1:8200/v1/sys/init"

		payload := strings.NewReader("{\n\t\"stored_shares\": 3,\n\t\"recovery_threshold\": 3,\n\t\"recovery_shares\": 5\n}")

		req, err := http.NewRequest("POST", url, payload)
		if err != nil {
			log.Panic().Err(err).Msg("")
		}

		req.Header.Add("Content-Type", "application/json")

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Error().Err(err).Msg("error in Do http client request")
		}

		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Panic().Err(err).Msg("")
		}

		log.Info().Msg(string(body))

		vaultResponse := VaultUnsealResponse{}
		err = json.Unmarshal(body, &vaultResponse)
		if err != nil {
			log.Panic().Err(err).Msg("")
		}

		viper.Set("vault.token", vaultResponse.RootToken)
		viper.Set("vault.unseal-keys", vaultResponse)
		viper.WriteConfig()
	} else {
		log.Info().Msg("vault token already exists, continuing")
	}
}

func waitGitlabToBeReady(dryRun bool) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, waitVaultToBeRunning skipped.")
		return
	}
	config := configs.ReadConfig()
	x := 50
	for i := 0; i < x; i++ {
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "namespace/gitlab")
		if err != nil {
			log.Error().Err(err).Msg("Waiting gitlab namespace to be born")
			time.Sleep(10 * time.Second)
		} else {
			log.Info().Msg("gitlab namespace found, continuing")
			time.Sleep(5 * time.Second)
			break
		}
	}
	x = 50
	for i := 0; i < x; i++ {
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "gitlab", "get", "pods", "-l", "app=webservice")
		if err != nil {
			log.Warn().Err(err).Msg("waiting gitlab pods to be born")
			time.Sleep(10 * time.Second)
		} else {
			log.Info().Msg("gitlab pods found, continuing")
			time.Sleep(15 * time.Second)
			break
		}
	}

}

// Notify user in the STOUT and also logfile
func informUser(message string, silentMode bool) {
	// if in silent mode, send message to the screen
	// silent mode will silent most of the messages, this function is not frequently called
	if silentMode {
		_, err := fmt.Fprintln(os.Stdout, message)
		if err != nil {
			log.Error().Err(err).Msg("")
		}
		return
	}
	log.Info().Msg(message)
	progressPrinter.LogMessage(fmt.Sprintf("- %s", message))
}
