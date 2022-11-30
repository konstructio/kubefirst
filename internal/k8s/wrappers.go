package k8s

import (
	"log"
	"sync"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// OpenPortForwardForLocal is a wrapper function to instantiate the necessary resources for Kubefirst
// console. OpenPortForwardForLocal receives channels as arguments, when this channels are closed, the
// port forwards are also closed.
//
// Every port forward that is open, is open in a Go routine, the function exists when all the (wg.Add(x)) x Go
// routines are done.
func OpenPortForwardForLocal(
	vaultStopChannel chan struct{},
	argoStopChannel chan struct{},
	argoCDStopChannel chan struct{},
	chartmuseumStopChannel chan struct{},
	minioStopChannel chan struct{},
	minioConsoleStopChannel chan struct{},
	kubefirstConsoleStopChannel chan struct{},
	AtlantisStopChannel chan struct{},
	MetaphorFrontendDevelopmentStopChannel chan struct{},
	MetaphorGoDevelopmentStopChannel chan struct{},
	MetaphorDevelopmentStopChannel chan struct{},
) error {

	var wg sync.WaitGroup
	wg.Add(8)

	// Vault
	go func() {
		OpenPortForwardPodWrapper(pkg.VaultPodName, pkg.VaultNamespace, pkg.VaultPodPort, pkg.VaultPodLocalPort, vaultStopChannel)
		wg.Done()
	}()

	// Argo
	go func() {
		OpenPortForwardPodWrapper(pkg.ArgoPodName, pkg.ArgoNamespace, pkg.ArgoPodPort, pkg.ArgoPodLocalPort, argoStopChannel)
		wg.Done()
	}()

	// ArgoCD
	go func() {
		OpenPortForwardPodWrapper(pkg.ArgoCDPodName, pkg.ArgoCDNamespace, pkg.ArgoCDPodPort, pkg.ArgoCDPodLocalPort, argoCDStopChannel)
		wg.Done()
	}()

	// chartmuseum
	go func() {
		OpenPortForwardPodWrapper(pkg.ChartmuseumPodName, pkg.ChartmuseumNamespace, pkg.ChartmuseumPodPort, pkg.ChartmuseumPodLocalPort, chartmuseumStopChannel)
		wg.Done()
	}()

	// Minio
	go func() {
		OpenPortForwardPodWrapper(pkg.MinioPodName, pkg.MinioNamespace, pkg.MinioPodPort, pkg.MinioPodLocalPort, minioStopChannel)
		wg.Done()
	}()

	// Minio Console
	go func() {
		OpenPortForwardPodWrapper(pkg.MinioConsolePodName, pkg.MinioConsoleNamespace, pkg.MinioConsolePodPort, pkg.MinioConsolePodLocalPort, minioConsoleStopChannel)
		wg.Done()
	}()

	// Kubefirst console
	go func() {
		OpenPortForwardPodWrapper(pkg.KubefirstConsolePodName, pkg.KubefirstConsoleNamespace, pkg.KubefirstConsolePodPort, pkg.KubefirstConsolePodLocalPort, kubefirstConsoleStopChannel)
		wg.Done()
	}()

	// Atlantis
	go func() {
		OpenPortForwardPodWrapper(pkg.AtlantisPodName, pkg.AtlantisNamespace, pkg.AtlantisPodPort, pkg.AtlantisPodLocalPort, AtlantisStopChannel)
		wg.Done()
	}()

	// MetaphorFrontendDevelopment
	go func() {
		OpenPortForwardServiceWrapper(pkg.MetaphorFrontendDevelopmentServiceName, pkg.MetaphorFrontendDevelopmentNamespace, pkg.MetaphorFrontendDevelopmentServicePort, pkg.MetaphorFrontendDevelopmentServiceLocalPort, MetaphorFrontendDevelopmentStopChannel)
		wg.Done()
	}()

	// MetaphorGoDevelopment
	go func() {
		OpenPortForwardServiceWrapper(pkg.MetaphorGoDevelopmentServiceName, pkg.MetaphorGoDevelopmentNamespace, pkg.MetaphorGoDevelopmentServicePort, pkg.MetaphorGoDevelopmentServiceLocalPort, MetaphorGoDevelopmentStopChannel)
		wg.Done()
	}()

	// MetaphorDevelopment
	go func() {
		OpenPortForwardServiceWrapper(pkg.MetaphorDevelopmentServiceName, pkg.MetaphorDevelopmentNamespace, pkg.MetaphorDevelopmentServicePort, pkg.MetaphorDevelopmentServiceLocalPort, MetaphorDevelopmentStopChannel)
		wg.Done()
	}()

	wg.Wait()
	return nil
}

// OpenPortForwardWrapper wrapper for PortForwardPod function. This functions make it easier to open and close port
// forward request. By providing the function parameters, the function will manage to create the port forward. The
// parameter for the stopChannel controls when the port forward must be closed.
//
// Example:
//
//	vaultStopChannel := make(chan struct{}, 1)
//	go func() {
//		OpenPortForwardWrapper(
//			pkg.VaultPodName,
//			pkg.VaultNamespace,
//			pkg.VaultPodPort,
//			pkg.VaultPodLocalPort,
//			vaultStopChannel)
//		wg.Done()
//	}()
func OpenPortForwardPodWrapper(podName string, namespace string, podPort int, podLocalPort int, stopChannel chan struct{}) {

	config1 := configs.ReadConfig()
	kubeconfig := config1.KubeConfigPath
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Println(err)
	}

	// readyCh communicate when the port forward is ready to get traffic
	readyCh := make(chan struct{})

	// todo: constants for podName, PodPort and localPort, namespace

	portForwardRequest := PortForwardAPodRequest{
		RestConfig: cfg,
		Pod: v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: namespace,
			},
		},
		PodPort:   podPort,
		LocalPort: podLocalPort,
		StopCh:    stopChannel,
		ReadyCh:   readyCh,
	}

	clientset, err := GetClientSet(false)

	go func() {
		err = PortForwardPod(clientset, portForwardRequest)
		if err != nil {
			log.Println(err)
		}
	}()

	select {
	case <-stopChannel:
		log.Println("leaving...")
		close(stopChannel)
		close(readyCh)
		break
	case <-readyCh:
		log.Println("port forwarding is ready to get traffic")
	}

	log.Printf("Pod %q at namespace %q has port-forward accepting local connections at port %d\n", podName, namespace, podLocalPort)
	//<-stopChannel
	return
}

func OpenPortForwardServiceWrapper(serviceName string, namespace string, servicePort int, serviceLocalPort int, stopChannel chan struct{}) {

	config1 := configs.ReadConfig()
	kubeconfig := config1.KubeConfigPath
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Println(err)
	}

	// readyCh communicate when the port forward is ready to get traffic
	readyCh := make(chan struct{})

	// todo: constants for podName, PodPort and localPort, namespace

	portForwardRequest := PortForwardAServiceRequest{
		RestConfig: cfg,
		Service: v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: namespace,
			},
		},
		ServicePort: servicePort,
		LocalPort:   serviceLocalPort,
		StopCh:      stopChannel,
		ReadyCh:     readyCh,
	}

	clientset, err := GetClientSet(false)

	go func() {
		err = PortForwardService(clientset, portForwardRequest)
		if err != nil {
			log.Println(err)
		}
	}()

	select {
	case <-stopChannel:
		log.Println("leaving...")
		close(stopChannel)
		close(readyCh)
		break
	case <-readyCh:
		log.Println("port forwarding is ready to get traffic")
	}

	log.Printf("Service %q at namespace %q has port-forward accepting local connections at port %d\n", serviceName, namespace, serviceLocalPort)
	//<-stopChannel
	return
}

func CreateSecretsFromCertificatesForLocalWrapper(config *configs.Config) error {

	for _, app := range pkg.GetCertificateAppList() {

		certFileName := config.MkCertPemFilesPath + app.AppName + "-cert.pem" // example: app-name-cert.pem
		keyFileName := config.MkCertPemFilesPath + app.AppName + "-key.pem"   // example: app-name-key.pem

		log.Printf("creating TLS k8s secret for %s...", app.AppName)

		// open file content
		certContent, err := pkg.GetFileContent(certFileName)
		if err != nil {
			return err
		}

		keyContent, err := pkg.GetFileContent(keyFileName)
		if err != nil {
			return err
		}

		data := make(map[string][]byte)
		data["tls.crt"] = certContent
		data["tls.key"] = keyContent

		// save content into secret
		err = CreateSecret(app.Namespace, app.AppName+"-tls", data)
		if err != nil {
			log.Println(err)
		}

		log.Printf("creating TLS k8s secret for %s done", app.AppName)
	}

	return nil
}
