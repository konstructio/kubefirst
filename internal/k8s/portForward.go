package k8s

import (
	"context"
	"errors"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
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

// todo: this is temporary
func OpenPortForwardForCloudConConsole() error {
	var wg sync.WaitGroup
	wg.Add(1)

	// Cloud Console UI
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

// deprecated
// OpenPortForwardForKubeConConsole was deprecated by OpenPortForwardForLocal, that handles port forwards using k8s
// go client.
func OpenPortForwardForKubeConConsole() error {

	var wg sync.WaitGroup
	wg.Add(8)
	// argo workflows
	go func() {
		output, err := PortForward(false, "argo", "svc/argo-server", "2746:2746")
		if err != nil {
			log.Println("error opening Argo Workflows port forward")
		}
		stderr := fmt.Sprint(output.Stderr)
		if len(stderr) > 0 {
			log.Println(stderr)
		}
		wg.Done()
	}()
	// argocd
	go func() {
		output, err := PortForward(false, "argocd", "svc/argocd-server", "8080:80")
		if err != nil {
			log.Println("error opening ArgoCD port forward")
		}
		stderr := fmt.Sprint(output.Stderr)
		if len(stderr) > 0 {
			log.Println(stderr)
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
		output, err := PortForward(false, "chartmuseum", "svc/chartmuseum", "8181:8080")
		if err != nil {
			log.Println("error opening Chartmuseum port forward")
		}
		stderr := fmt.Sprint(output.Stderr)
		if len(stderr) > 0 {
			log.Println(stderr)
		}
		wg.Done()
	}()

	// vault
	go func() {
		output, err := PortForward(false, "vault", "svc/vault", "8200:8200")
		if err != nil {
			log.Println("error opening Vault port forward")
		}
		stderr := fmt.Sprint(output.Stderr)
		if len(stderr) > 0 {
			log.Println(stderr)
		}

		wg.Done()
	}()

	// minio
	go func() {
		output, err := PortForward(false, "minio", "svc/minio", "9000:9000")
		if err != nil {
			log.Println("error opening Minio port forward")
		}
		stderr := fmt.Sprint(output.Stderr)
		if len(stderr) > 0 {
			log.Println(stderr)
		}
		wg.Done()
	}()

	// minio console
	go func() {
		output, err := PortForward(false, "minio", "svc/minio", "9000:9000")
		if err != nil {
			log.Println("error opening Minio port forward")
		}
		stderr := fmt.Sprint(output.Stderr)
		if len(stderr) > 0 {
			log.Println(stderr)
		}
		wg.Done()
	}()

	// minio console
	go func() {
		output, err := PortForward(false, "minio", "svc/minio-console", "9001:9001")
		if err != nil {
			log.Println("error opening Minio-console port forward")
		}
		stderr := fmt.Sprint(output.Stderr)
		if len(stderr) > 0 {
			log.Println(stderr)
		}
		wg.Done()
	}()

	// Kubecon console ui
	go func() {
		output, err := PortForward(false, "kubefirst", "svc/kubefirst-console", "9094:80")
		if err != nil {
			log.Println("error opening Kubefirst-console port forward")
		}
		stderr := fmt.Sprint(output.Stderr)
		if len(stderr) > 0 {
			log.Println(stderr)
		}
		wg.Done()
	}()

	wg.Wait()

	return nil
}

// OpenAtlantisPortForward opens port forward for Atlantis
func OpenAtlantisPortForward() error {

	output, err := PortForward(false, "atlantis", "svc/atlantis", "4141:80")
	if err != nil {
		return errors.New("error opening Atlantis port forward")
	}
	stderr := fmt.Sprint(output.Stderr)
	if len(stderr) > 0 {
		return errors.New(stderr)
	}

	return nil
}
