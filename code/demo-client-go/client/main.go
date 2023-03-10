package main

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	TestRestClient()
	TestClientset()
	TestDynamicClient()
	TestDiscoveryClient()
}
func TestClientset() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	pod, err := clientset.CoreV1().Pods("default").Get(context.TODO(), "nginx", v1.GetOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Println(pod.Name)
}
func TestRestClient() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err.Error())
	}
	config.GroupVersion = &corev1.SchemeGroupVersion
	config.APIPath = "/api"
	config.NegotiatedSerializer = scheme.Codecs

	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		panic(err)
	}
	pod := corev1.Pod{}
	err = restClient.Get().Namespace("kube-system").Resource("pods").Name("traefik-d497b4cb6-229vw").Do(context.TODO()).Into(&pod)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(pod.Name)
	}
}

func TestDynamicClient() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err)
	}
	dynamicClientset, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	unstructured, err := dynamicClientset.Resource(schema.GroupVersionResource{Version: "v1", Resource: "pods"}).Namespace("kube-system").Get(context.TODO(), "traefik-d497b4cb6-229vw", v1.GetOptions{})
	pod := corev1.Pod{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), &pod)
	if err != nil {
		panic(err)
	}
	fmt.Println(pod.Name)
}

func TestDiscoveryClient() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err)
	}
	client, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		panic(err)
	}

	apiGroup, apiResourceList, err := client.ServerGroupsAndResources()
	if err != nil {
		panic(err)
	}
	// ??????Group??????
	fmt.Printf("APIGroup :\n\n %v\n\n\n\n", apiGroup)
	// APIResourceListSlice????????????????????????????????????????????????GroupVersion????????????
	for _, singleAPIResourceList := range apiResourceList {
		// GroupVersion????????????????????????"apps/v1"
		groupVerionStr := singleAPIResourceList.GroupVersion
		// ParseGroupVersion????????????????????????????????????
		gv, err := schema.ParseGroupVersion(groupVerionStr)
		if err != nil {
			panic(err)
		}
		fmt.Println("*****************************************************************")
		fmt.Printf("GV string [%v]\nGV struct [%#v]\nresources :\n\n", groupVerionStr, gv)
		// APIResources????????????????????????????????????GroupVersion??????????????????
		for _, singleAPIResource := range singleAPIResourceList.APIResources {
			fmt.Printf("%v\n", singleAPIResource.Name)
		}
	}
}
