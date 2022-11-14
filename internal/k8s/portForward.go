package k8s

import (
	"context"
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type PortForwardAPodRequest struct {
	// RestConfig is the kubernetes config
	RestConfig *rest.Config
	// Pod is the selected pod for this port forwarding
	Pod v1.Pod
	// LocalPort is the local port that will be selected to expose the PodPort
	LocalPort int
	// PodPort is the target port for the pod
	PodPort int

	//// Steams configures where to write or read input from
	//Streams genericclioptions.IOStreams

	// StopCh is the channel used to manage the port forward lifecycle
	StopCh <-chan struct{}
	// ReadyCh communicates when the tunnel is ready to receive traffic
	ReadyCh chan struct{}
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
func OpenPortForwardWrapper(podName string, namespace string, podPort int, podLocalPort int, stopChannel chan struct{}) {

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

	log.Printf("Pod %q at namespace %q has port forward accepting connections at port %d\n", podName, namespace, podLocalPort)
	//<-stopChannel

	return
}

// PortForwardPod receives a PortForwardAPodRequest, and enable port forward for the specified resource. If the provided
// Pod name matches a running Pod, it will try to port forward for that Pod on the specified port.
func PortForwardPod(clientset *kubernetes.Clientset, req PortForwardAPodRequest) error {

	podList, err := clientset.CoreV1().Pods(req.Pod.Namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil || len(podList.Items) == 0 {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var runningPod *v1.Pod
	for _, pod := range podList.Items {
		// pick the first pod found to be running
		if pod.Status.Phase == v1.PodRunning && strings.HasPrefix(pod.Name, req.Pod.Name) {
			runningPod = &pod
			break
		}
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", runningPod.Namespace, runningPod.Name)
	hostIP := strings.TrimLeft(req.RestConfig.Host, "htps:/")

	transport, upgrader, err := spdy.RoundTripperFor(req.RestConfig)
	if err != nil {
		return err
	}

	dialer := spdy.NewDialer(
		upgrader,
		&http.Client{Transport: transport},
		http.MethodPost,
		&url.URL{
			Scheme: "https",
			Path:   path,
			Host:   hostIP,
		},
	)

	fw, err := portforward.New(
		dialer,
		[]string{fmt.Sprintf(
			"%d:%d",
			req.LocalPort,
			req.PodPort)},
		req.StopCh,
		req.ReadyCh,
		os.Stdout,
		os.Stderr)
	if err != nil {
		return err
	}

	err = fw.ForwardPorts()
	if err != nil {
		return err
	}

	return nil

}
