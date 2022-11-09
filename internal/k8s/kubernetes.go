package k8s

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/itchyny/gojq"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

var gitlabToolboxPodName string

var GitlabSecretClient coreV1Types.SecretInterface

type secret struct {
	namespace string
	name      string
}

type PatchJson struct {
	Op   string `json:"op"`
	Path string `json:"path"`
}

func GetPodNameByLabel(podsClient coreV1Types.PodInterface, label string) string {
	pods, err := podsClient.List(context.TODO(), metaV1.ListOptions{LabelSelector: label})
	if err != nil {
		log.Println(err)
	}

	gitlabToolboxPodName = pods.Items[0].Name

	return gitlabToolboxPodName
}

func DeletePodByLabel(podsClient coreV1Types.PodInterface, label string) {
	err := podsClient.DeleteCollection(context.TODO(), metaV1.DeleteOptions{}, metaV1.ListOptions{LabelSelector: label})
	if err != nil {
		log.Println(err)
	} else {
		log.Printf("Success delete of pods with label(%s).", label)
	}
}

func GetSecretValue(k8sClient coreV1Types.SecretInterface, secretName, key string) string {
	secret, err := k8sClient.Get(context.TODO(), secretName, metaV1.GetOptions{})
	if err != nil {
		log.Println(fmt.Sprintf("error getting key: %s from secret: %s", key, secretName), err)
	}
	return string(secret.Data[key])
}

func DeleteRegistryApplication(skipDeleteRegistryApplication bool) {

	if !skipDeleteRegistryApplication {

		log.Println("refreshing argocd session token")
		argocd.GetArgocdAuthToken(false)

		url := "https://localhost:8080/api/v1/applications/registry"
		_, _, err := pkg.ExecShellReturnStrings("curl", "-k", "-vL", "-X", "DELETE", url, "-H", fmt.Sprintf("Authorization: Bearer %s", viper.GetString("argocd.admin.apitoken")))
		if err != nil {
			log.Panicf("error: delete registry applicatoin from argocd failed: %s", err)
		}
		log.Println("waiting for argocd deletion to complete")
		time.Sleep(300 * time.Second)
	} else {
		log.Println("skip:  deleteRegistryApplication")
	}
}

func GetResourcesDynamically(dynamic dynamic.Interface,
	ctx context.Context,
	group string,
	version string,
	resource string,
	namespace string) (
	[]unstructured.Unstructured, error) {

	resourceId := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}
	list, err := dynamic.Resource(resourceId).Namespace(namespace).
		List(ctx, metaV1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

func GetResourcesByJq(dynamic dynamic.Interface, ctx context.Context, group string,
	version string, resource string, namespace string, jq string) (
	[]unstructured.Unstructured, error) {

	resources := make([]unstructured.Unstructured, 0)

	query, err := gojq.Parse(jq)
	if err != nil {
		return nil, err
	}

	items, err := GetResourcesDynamically(dynamic, ctx, group, version, resource, namespace)
	if err != nil {
		return nil, err
	}

	for _, item := range items {

		// Convert object to raw JSON
		var rawJson interface{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &rawJson)
		if err != nil {
			return nil, err
		}

		// Evaluate jq against JSON
		iter := query.Run(rawJson)
		for {
			result, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok := result.(error); ok {
				if err != nil {
					return nil, err
				}
			} else {
				boolResult, ok := result.(bool)
				if !ok {
					fmt.Println("Query returned non-boolean value")
				} else if boolResult {
					resources = append(resources, item)
				}
			}
		}
	}
	return resources, nil
}

// GetClientSet - Get reference to k8s credentials to use APIS
func GetClientSet(dryRun bool) (*kubernetes.Clientset, error) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, GetClientSet skipped.")
		return nil, nil
	}
	config := configs.ReadConfig()

	kubeconfig, err := clientcmd.BuildConfigFromFlags("", config.KubeConfigPath)
	if err != nil {
		log.Printf("Error getting kubeconfig: %s", err)
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		log.Printf("Error getting clientset: %s", err)
		return clientset, err
	}

	return clientset, nil
}

// PortForward - opens port-forward to services
func PortForward(dryRun bool, namespace string, filter string, ports string) (*exec.Cmd, error) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, K8sPortForward skipped.")
		return nil, nil
	}
	config := configs.ReadConfig()

	var kPortForwardOutb, kPortForwardErrb bytes.Buffer
	kPortForward := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", namespace, "port-forward", filter, ports)
	kPortForward.Stdout = &kPortForwardOutb
	kPortForward.Stderr = &kPortForwardErrb
	err := kPortForward.Start()

	// make port forward port available for log
	log.Printf("kubectl port-forward started for (%s) available at http://localhost:%s", filter, strings.Split(ports, ":")[0])
	//defer kPortForwardVault.Process.Signal(syscall.SIGTERM)

	//Please, don't remove this sleep, pf takes a while to be ready to search calls.
	//So, if next command is called to curl this address it will get connection refused.
	//this sleep protects that.
	//Please, don't remove this comment either.
	time.Sleep(time.Second * 5)
	log.Println(config.KubectlClientPath, " ", "--kubeconfig", " ", config.KubeConfigPath, " ", "-n", " ", namespace, " ", "port-forward", " ", filter, " ", ports)
	if err != nil {
		// If it doesn't error, we kinda don't care much.
		log.Printf("Commad Execution STDOUT: %s", kPortForwardOutb.String())
		log.Printf("Commad Execution STDERR: %s", kPortForwardErrb.String())
		log.Printf("error: failed to port-forward to %s in main thread %s", filter, err)
		return kPortForward, err
	}

	return kPortForward, nil
}

func WaitForNamespaceandPods(dryRun bool, config *configs.Config, namespace, podLabel string) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, WaitForNamespaceandPods skipped")
		return
	}
	if !viper.GetBool("create.softserve.ready") {
		x := 50
		for i := 0; i < x; i++ {
			_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", namespace, "get", fmt.Sprintf("namespace/%s", namespace))
			if err != nil {
				log.Println(fmt.Sprintf("waiting for %s namespace to create ", namespace))
				time.Sleep(10 * time.Second)
			} else {
				log.Println(fmt.Sprintf("namespace %s found, continuing", namespace))
				time.Sleep(10 * time.Second)
				i = 51
			}
		}
		for i := 0; i < x; i++ {
			_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", namespace, "get", "pods", "-l", podLabel)
			if err != nil {
				log.Println(fmt.Sprintf("waiting for %s pods to create ", namespace))
				time.Sleep(10 * time.Second)
			} else {
				log.Println(fmt.Sprintf("%s pods found, continuing", namespace))
				time.Sleep(10 * time.Second)
				break
			}
		}
		viper.Set("create.softserve.ready", true)
		viper.WriteConfig()
	} else {
		log.Println("soft-serve is ready, skipping")
	}
}

func PatchSecret(k8sClient coreV1Types.SecretInterface, secretName, key, val string) {
	secret, err := k8sClient.Get(context.TODO(), secretName, metaV1.GetOptions{})
	if err != nil {
		log.Println(fmt.Sprintf("error getting key: %s from secret: %s", key, secretName), err)
	}
	secret.Data[key] = []byte(val)
	k8sClient.Update(context.TODO(), secret, metaV1.UpdateOptions{})
}

func CreateVaultConfiguredSecret(dryRun bool, config *configs.Config) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, CreateVaultConfiguredSecret skipped.")
		return
	}
	if !viper.GetBool("vault.configuredsecret") {
		var output bytes.Buffer
		// todo - https://github.com/bcreane/k8sutils/blob/master/utils.go
		// kubectl create secret generic vault-configured --from-literal=isConfigured=true
		// the purpose of this command is to let the vault-unseal Job running in kuberenetes know that external secrets store should be able to connect to the configured vault
		k := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "vault", "create", "secret", "generic", "vault-configured", "--from-literal=isConfigured=true")
		k.Stdout = &output
		k.Stderr = os.Stderr
		err := k.Run()
		if err != nil {
			log.Panicf("failed to create secret for vault-configured: %s", err)
		}
		log.Printf("the secret create output is: %s", output.String())

		viper.Set("vault.configuredsecret", true)
		viper.WriteConfig()
	} else {
		log.Println("vault secret already created")
	}
}

func WaitForGitlab(dryRun bool, config *configs.Config) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, WaitForGitlab skipped.")
		return
	}
	var output bytes.Buffer
	// todo - add a viper.GetBool() check to the beginning of this function
	// todo write in golang? see here -> https://github.com/bcreane/k8sutils/blob/master/utils.go
	k := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "gitlab", "wait", "--for=condition=ready", "pod", "-l", "app=webservice", "--timeout=300s")
	k.Stdout = &output
	k.Stderr = os.Stderr
	err := k.Run()
	if err != nil {
		log.Panicf("failed to execute kubectl wait for gitlab pods with label app=webservice: %s \n%s", output.String(), err)
	}
	log.Printf("the output is: %s", output.String())
}

func RemoveSelfSignedCertArgoCD(argocdPodClient coreV1Types.PodInterface) error {
	log.Printf("Removing Self-Signed Certificate from argocd-secret")

	log.Printf("Removing tls.crt")
	err := clearSecretField("argocd", "argocd-secret", "/data/tls.crt")
	if err != nil {
		log.Printf("err removing tls.crt from argo-secret: %s", err)
		return err
	}

	log.Printf("Removing tls.key")
	err = clearSecretField("argocd", "argocd-secret", "/data/tls.key")
	if err != nil {
		log.Printf("err removing tls.crt from argo-secret: %s", err)
		return err
	}

	// delete argocd-server pod to pickup the new cert-manager cert if ready
	DeletePodByLabel(argocdPodClient, "app.kubernetes.io/name=argocd-server")

	return nil
}

// remove field from k8s secret using sdk
func clearSecretField(namespace, name, field string) error {
	log.Printf("Prepare secret to be patched: ns: %s name: %s path: %s", namespace, name, field)
	secret := secret{
		namespace: namespace,
		name:      name,
	}

	payload := []PatchJson{{
		Op:   "remove",
		Path: field,
	}}

	clientset, err := GetClientSet(false)
	if err != nil {
		log.Printf("Error creating k8s clientset : %s", err)
		return err
	}

	err = secret.patchSecret(clientset, payload)
	if err != nil {
		log.Printf("Error calling patchSecret : %s", err)
		return err
	}
	return nil
}

func (p *secret) patchSecret(k8sClient *kubernetes.Clientset, payload []PatchJson) error {

	payloadBytes, _ := json.Marshal(payload)

	log.Printf("Patching secret on K8S via SDK")
	_, err := k8sClient.CoreV1().Secrets(p.namespace).Patch(context.TODO(), p.name, k8sTypes.JSONPatchType, payloadBytes, metaV1.PatchOptions{})

	if err != nil {
		log.Printf("Error patching secret : %s", err)
		return err
	}
	return nil
}

// todo: deprecate the other functions
func LoopUntilPodIsReady(dryRun bool) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, loopUntilPodIsReady skipped.")
		return
	}
	token := viper.GetString("vault.token")
	if len(token) == 0 {

		totalAttempts := 50
		url := "http://localhost:8200/v1/sys/health"
		for i := 0; i < totalAttempts; i++ {
			log.Printf("vault is not ready yet, sleeping and checking again, attempt (%d/%d)", i+1, totalAttempts)
			time.Sleep(10 * time.Second)

			req, _ := http.NewRequest("GET", url, nil)

			req.Header.Add("Content-Type", "application/json")

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Println("error with http request Do, vault is not available", err)
				// todo: temporary code
				log.Println("trying to open port-forward again...")
				go func() {
					_, err := PortForward(false, "vault", "svc/vault", "8200:8200")
					if err != nil {
						log.Println("error opening Vault port forward")
					}
				}()
				continue
			}

			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			if err != nil {
				log.Println("vault is available but the body is not what is expected ", err)
				continue
			}

			var responseJson map[string]interface{}

			if err := json.Unmarshal(body, &responseJson); err != nil {
				log.Printf("vault is available but the body is not what is expected %s", err)
				continue
			}

			_, ok := responseJson["initialized"]
			if ok {
				log.Printf("vault is initialized and is in the expected state")
				return
			}
			log.Panic("vault was never initialized")
		}
		viper.Set("vault.status.running", true)
		viper.WriteConfig()
	} else {
		log.Println("vault token already exists, skipping vault health checks loopUntilPodIsReady")
	}
}

// todo: deprecate the other functions
func SetArgocdCreds(dryRun bool) {
	if dryRun {
		log.Printf("[#99] Dry-run mode, setArgocdCreds skipped.")
		viper.Set("argocd.admin.password", "dry-run-not-real-pwd")
		viper.Set("argocd.admin.username", "dry-run-not-admin")
		viper.WriteConfig()
		return
	}
	clientset, err := GetClientSet(dryRun)
	if err != nil {
		panic(err.Error())
	}
	argocd.ArgocdSecretClient = clientset.CoreV1().Secrets("argocd")

	argocdPassword := GetSecretValue(argocd.ArgocdSecretClient, "argocd-initial-admin-secret", "password")
	if argocdPassword == "" {
		log.Panicf("Missing argocdPassword")
	}

	viper.Set("argocd.admin.password", argocdPassword)
	viper.Set("argocd.admin.username", "admin")
	viper.WriteConfig()
}

// todo: this is temporary
func OpenPortForwardForKubeConConsole() error {

	var wg sync.WaitGroup
	wg.Add(8)
	// argo workflows
	go func() {
		_, err := PortForward(false, "argo", "svc/argo-server", "2746:2746")
		if err != nil {
			log.Println("error opening Argo Workflows port forward")
		}
		wg.Done()
	}()
	// argocd
	go func() {
		_, err := PortForward(false, "argocd", "svc/argocd-server", "8080:80")
		if err != nil {
			log.Println("error opening ArgoCD port forward")
		}
		wg.Done()
	}()

	// atlantis
	go func() {
		err := OpenAtlantisPortForward()
		if err != nil {
			log.Println(err)
		}
		wg.Done()
	}()

	// chartmuseum
	go func() {
		_, err := PortForward(false, "chartmuseum", "svc/chartmuseum", "8181:8080")
		if err != nil {
			log.Println("error opening Chartmuseum port forward")
		}
		wg.Done()
	}()

	// vault
	go func() {
		_, err := PortForward(false, "vault", "svc/vault", "8200:8200")
		if err != nil {
			log.Println("error opening Vault port forward")
		}
		wg.Done()
	}()

	// minio
	go func() {
		_, err := PortForward(false, "minio", "svc/minio", "9000:9000")
		if err != nil {
			log.Println("error opening Minio port forward")
		}
		wg.Done()
	}()

	// minio console
	go func() {
		_, err := PortForward(false, "minio", "svc/minio-console", "9001:9001")
		if err != nil {
			log.Println("error opening Minio-console port forward")
		}
		wg.Done()
	}()

	// Kubecon console ui
	go func() {
		_, err := PortForward(false, "kubefirst", "svc/kubefirst-console", "9094:80")
		if err != nil {
			log.Println("error opening Kubefirst-console port forward")
		}
		wg.Done()
	}()

	wg.Wait()

	return nil
}

// OpenAtlantisPortForward opens port forward for Atlantis
func OpenAtlantisPortForward() error {

	_, err := PortForward(false, "atlantis", "svc/atlantis", "4141:80")
	if err != nil {
		return errors.New("error opening Atlantis port forward")
	}
	return nil
}
