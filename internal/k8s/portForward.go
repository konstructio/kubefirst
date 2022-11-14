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

func OpenManagedPortForward(podName string, namespace string, podPort int, podLocalPort int, stopChannel chan struct{}) {

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
	fmt.Println("---debug---")
	fmt.Println(portForwardRequest.PodPort)
	fmt.Println(portForwardRequest.LocalPort)
	fmt.Println(portForwardRequest.Pod.Namespace)
	fmt.Println(portForwardRequest.Pod.Name)
	fmt.Println("---debug---")

	clientset, err := GetClientSet(false)

	go func() {
		err = PortForwardAKubefirstPod(clientset, portForwardRequest)
		if err != nil {
			log.Println(err)
		}
	}()

	fmt.Println("Port forwarding is ready to get traffic. have fun!")

	select {
	case <-stopChannel:
		fmt.Println("leaving...")
		close(stopChannel)
		close(readyCh)
		break
	case <-readyCh:
		fmt.Println("accepting connections")
	}

	fmt.Printf("Pod %q at namespace %q has port forward accepting connections at port %d\n", podName, namespace, podLocalPort)
	//<-stopChannel

	return
}

func PortForwardAPod(req PortForwardAPodRequest) error {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", req.Pod.Namespace, req.Pod.Name)
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

	return fw.ForwardPorts()
}

func PortForwardAKubefirstPod(clientset *kubernetes.Clientset, req PortForwardAPodRequest) error {

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
	fmt.Println("---debug2---")
	fmt.Println(path + hostIP)
	fmt.Println("---debug2---")

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
