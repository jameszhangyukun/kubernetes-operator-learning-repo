package main

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	TestRestClient()
	TestClientset()
}
func TestClientset()  {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil{
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
	}else {
		fmt.Println(pod.Name)
	}
}

