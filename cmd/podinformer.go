/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	//"k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

//https://firehydrant.com/blog/stay-informed-with-kubernetes-informers/
// podinformerCmd represents the podinformer command
// this doesn't use logs, as the goal is to send to kubectl logs..(stdout)
var podinformerCmd = &cobra.Command{
	Use:   "podinformer",
	Short: "To be used in-cluster",
	Long:  `TBD`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("podinformer called")
		watchNamespace()
	},
}

func init() {
	rootCmd.AddCommand(podinformerCmd)

}

func watchNamespace() {
	clientSet := getK8SConfig()
	factory := informers.NewSharedInformerFactory(clientSet, 0)
	informer := factory.Core().V1().Pods().Informer()
	stopper := make(chan struct{})
	defer close(stopper)
	interestingPods := make(chan string)
	defer close(interestingPods)
	go checkPod(interestingPods, stopper)
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// "k8s.io/apimachinery/pkg/apis/meta/v1" provides an Object
			// interface that allows us to get metadata easily
			mObj := obj.(*corev1.Pod)
			fmt.Printf("New Pod Added to Store: %s - %v", mObj.Name, mObj.ObjectMeta.Labels)
			fmt.Printf("\nNew Pod updated to Store -  status: %s", mObj.Status.Phase)
		},
		UpdateFunc: func(old, new interface{}) {
			// "k8s.io/apimachinery/pkg/apis/meta/v1" provides an Object
			// interface that allows us to get metadata easily
			//oldObj := old.(*corev1.Pod)
			newObj := new.(*corev1.Pod)
			//fmt.Printf("\nNew Pod updated to Store - old: %s", oldObj.GetName())
			//fmt.Printf("\nNew Pod updated to Store -  status: %s", newObj.Status.Phase)
			fmt.Printf("\nNew Pod updated  -  status: %s", newObj.Status.Phase)
			//fmt.Printf("\nNew Pod updated to Store - new: %s", newObj.GetName())
			if newObj.Status.Phase == corev1.PodRunning {
				fmt.Printf("\nNew Pod updated of interest found -  status: %s", newObj.Status.Phase)
				interestingPods <- newObj.GetName()
			}
		},
		DeleteFunc: func(obj interface{}) {
			// "k8s.io/apimachinery/pkg/apis/meta/v1" provides an Object
			// interface that allows us to get metadata easily
			mObj := obj.(*corev1.Pod)
			fmt.Printf("\nNew Pod deleted from Store: %s", mObj.GetName())
		},
	})
	informer.Run(stopper)
}

func getK8SConfig() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	fmt.Println("Config used")
	return clientSet

}

func checkPod(in <-chan string, stopper chan struct{}) {
	fmt.Println("Started Listener")
	var requiredPods map[string]bool
	requiredPods = make(map[string]bool)
	requiredPods["sample"] = false
	requiredPods["sample2"] = false
	fmt.Println(requiredPods)

	for {
		receivedPodName := <-in
		fmt.Printf("\nInteresting Pod: %s", receivedPodName)
		missingSomething := false
		if _, ok := requiredPods[receivedPodName]; ok {
			//do something here
			requiredPods[receivedPodName] = true
		}
		for key, value := range requiredPods {
			fmt.Println("Key:", key, "Value:", value)
			if value == false {
				missingSomething = true
			}
		}
		if !missingSomething {
			fmt.Println("All required objects found, ready to close waiting channels")
			fmt.Println(requiredPods)
			os.Exit(0)
		}
	}
}
