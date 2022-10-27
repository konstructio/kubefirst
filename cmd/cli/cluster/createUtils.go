package cluster

//
//import (
//	"encoding/json"
//	"io"
//	"log"
//	"net/http"
//	"strings"
//	"time"
//
//	"github.com/kubefirst/kubefirst/configs"
//	"github.com/kubefirst/kubefirst/internal/argocd"
//	"github.com/kubefirst/kubefirst/internal/k8s"
//	"github.com/kubefirst/kubefirst/pkg"
//	"github.com/spf13/viper"
//)
//
//// todo: move it to internals/ArgoCD
//func setArgocdCreds(dryRun bool) {
//	if dryRun {
//		log.Printf("[#99] Dry-run mode, setArgocdCreds skipped.")
//		viper.Set("argocd.admin.password", "dry-run-not-real-pwd")
//		viper.Set("argocd.admin.username", "dry-run-not-admin")
//		viper.WriteConfig()
//		return
//	}
//	clientset, err := k8s.GetClientSet(dryRun)
//	if err != nil {
//		panic(err.Error())
//	}
//	argocd.ArgocdSecretClient = clientset.CoreV1().Secrets("argocd")
//
//	argocdPassword := k8s.GetSecretValue(argocd.ArgocdSecretClient, "argocd-initial-admin-secret", "password")
//	if argocdPassword == "" {
//		log.Panicf("Missing argocdPassword")
//	}
//
//	viper.Set("argocd.admin.password", argocdPassword)
//	viper.Set("argocd.admin.username", "admin")
//	viper.WriteConfig()
//}
//
//func waitArgoCDToBeReady(dryRun bool) {
//	if dryRun {
//		log.Printf("[#99] Dry-run mode, waitArgoCDToBeReady skipped.")
//		return
//	}
//	config := configs.ReadConfig()
//	x := 50
//	for i := 0; i < x; i++ {
//		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "namespace/argocd")
//		if err != nil {
//			log.Println("Waiting argocd to be born")
//			time.Sleep(10 * time.Second)
//		} else {
//			log.Println("argocd namespace found, continuing")
//			time.Sleep(5 * time.Second)
//			break
//		}
//	}
//	for i := 0; i < x; i++ {
//		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argocd", "get", "pods", "-l", "app.kubernetes.io/name=argocd-server")
//		if err != nil {
//			log.Println("Waiting for argocd pods to create, checking in 10 seconds")
//			time.Sleep(10 * time.Second)
//		} else {
//			log.Println("argocd pods found, waiting for them to be running")
//			viper.Set("argocd.ready", true)
//			viper.WriteConfig()
//			time.Sleep(15 * time.Second)
//			break
//		}
//	}
//}
//
//func waitVaultToBeRunning(dryRun bool) {
//	if dryRun {
//		log.Printf("[#99] Dry-run mode, waitVaultToBeRunning skipped.")
//		return
//	}
//	token := viper.GetString("vault.token")
//	if len(token) == 0 {
//		config := configs.ReadConfig()
//		x := 50
//		for i := 0; i < x; i++ {
//			_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "namespace/vault")
//			if err != nil {
//				log.Println("Waiting vault to be born")
//				time.Sleep(10 * time.Second)
//			} else {
//				log.Println("vault namespace found, continuing")
//				time.Sleep(25 * time.Second)
//				break
//			}
//		}
//
//		//! failing
//		x = 50
//		for i := 0; i < x; i++ {
//			_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "vault", "get", "pods", "-l", "app.kubernetes.io/instance=vault")
//			if err != nil {
//				log.Println("Waiting vault pods to create")
//				time.Sleep(10 * time.Second)
//			} else {
//				log.Println("vault pods found, continuing")
//				time.Sleep(15 * time.Second)
//				break
//			}
//		}
//	} else {
//		log.Println("vault token arleady exists, skipping vault health checks waitVaultToBeRunning")
//	}
//}
//
//func loopUntilPodIsReady(dryRun bool) {
//	if dryRun {
//		log.Printf("[#99] Dry-run mode, loopUntilPodIsReady skipped.")
//		return
//	}
//	token := viper.GetString("vault.token")
//	if len(token) == 0 {
//
//		totalAttempts := 50
//		url := "http://localhost:8200/v1/sys/health"
//		for i := 0; i < totalAttempts; i++ {
//			log.Printf("vault is not ready yet, sleeping and checking again, attempt (%d/%d)", i+1, totalAttempts)
//			time.Sleep(10 * time.Second)
//
//			req, _ := http.NewRequest("GET", url, nil)
//
//			req.Header.Add("Content-Type", "application/json")
//
//			res, err := http.DefaultClient.Do(req)
//			if err != nil {
//				log.Println("error with http request Do, vault is not available", err)
//				continue
//			}
//
//			defer res.Body.Close()
//			body, err := io.ReadAll(res.Body)
//			if err != nil {
//				log.Println("vault is available but the body is not what is expected ", err)
//				continue
//			}
//
//			var responseJson map[string]interface{}
//
//			if err := json.Unmarshal(body, &responseJson); err != nil {
//				log.Printf("vault is available but the body is not what is expected %s", err)
//				continue
//			}
//
//			_, ok := responseJson["initialized"]
//			if ok {
//				log.Printf("vault is initialized and is in the expected state")
//				return
//			}
//			log.Panic("vault was never initialized")
//		}
//		viper.Set("vault.status.running", true)
//		viper.WriteConfig()
//	} else {
//		log.Println("vault token already exists, skipping vault health checks loopUntilPodIsReady")
//	}
//}
//
//type VaultInitResponse struct {
//	Initialized                bool   `json:"initialized"`
//	Sealed                     bool   `json:"sealed"`
//	Standby                    bool   `json:"standby"`
//	PerformanceStandby         bool   `json:"performance_standby"`
//	ReplicationPerformanceMode string `json:"replication_performance_mode"`
//	ReplicationDrMode          string `json:"replication_dr_mode"`
//	ServerTimeUtc              int    `json:"server_time_utc"`
//	Version                    string `json:"version"`
//}
//
//type VaultUnsealResponse struct {
//	UnsealKeysB64         []interface{} `json:"unseal_keys_b64"`
//	UnsealKeysHex         []interface{} `json:"unseal_keys_hex"`
//	UnsealShares          int           `json:"unseal_shares"`
//	UnsealThreshold       int           `json:"unseal_threshold"`
//	RecoveryKeysBase64    []string      `json:"recovery_keys_base64"`
//	RecoveryKeys          []string      `json:"recovery_keys"`
//	RecoveryKeysShares    int           `json:"recovery_keys_shares"`
//	RecoveryKeysThreshold int           `json:"recovery_keys_threshold"`
//	RootToken             string        `json:"root_token"`
//	Keys                  []string      `json:"keys"`
//	KeysB64               []string      `json:"keys_base64"`
//}
//
//func initializeVaultAndAutoUnseal(dryRun bool) {
//	if dryRun {
//		log.Printf("[#99] Dry-run mode, initializeVaultAndAutoUnseal skipped.")
//		return
//	}
//
//	token := viper.GetString("vault.token")
//	if len(token) == 0 {
//
//		time.Sleep(time.Second * 10)
//		url := "http://127.0.0.1:8200/v1/sys/init"
//
//		payload := strings.NewReader("{\n\t\"stored_shares\": 3,\n\t\"recovery_threshold\": 3,\n\t\"recovery_shares\": 5\n}")
//
//		req, err := http.NewRequest("POST", url, payload)
//		if err != nil {
//			log.Panic(err)
//		}
//
//		req.Header.Add("Content-Type", "application/json")
//
//		res, err := http.DefaultClient.Do(req)
//		if err != nil {
//			log.Println("error in Do http client request", err)
//		}
//
//		defer res.Body.Close()
//		body, err := io.ReadAll(res.Body)
//		if err != nil {
//			log.Panic(err)
//		}
//
//		log.Println(string(body))
//
//		vaultResponse := VaultUnsealResponse{}
//		err = json.Unmarshal(body, &vaultResponse)
//		if err != nil {
//			log.Panic(err)
//		}
//
//		viper.Set("vault.token", vaultResponse.RootToken)
//		viper.Set("vault.unseal-keys", vaultResponse)
//		viper.WriteConfig()
//	} else {
//		log.Println("vault token already exists, continuing")
//	}
//}
//
//func waitGitlabToBeReady(dryRun bool) {
//	if dryRun {
//		log.Printf("[#99] Dry-run mode, waitVaultToBeRunning skipped.")
//		return
//	}
//	config := configs.ReadConfig()
//	x := 50
//	for i := 0; i < x; i++ {
//		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "get", "namespace/gitlab")
//		if err != nil {
//			log.Println("Waiting gitlab namespace to be born")
//			time.Sleep(10 * time.Second)
//		} else {
//			log.Println("gitlab namespace found, continuing")
//			time.Sleep(5 * time.Second)
//			break
//		}
//	}
//	x = 50
//	for i := 0; i < x; i++ {
//		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "gitlab", "get", "pods", "-l", "app=webservice")
//		if err != nil {
//			log.Println("Waiting gitlab pods to be born")
//			time.Sleep(10 * time.Second)
//		} else {
//			log.Println("gitlab pods found, continuing")
//			time.Sleep(15 * time.Second)
//			break
//		}
//	}
//
//}
