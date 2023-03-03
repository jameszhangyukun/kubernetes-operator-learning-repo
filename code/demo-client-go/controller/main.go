package main

import (
	"github.jameszhangyukun.com/demo-client-go/controller/pkg"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main()  {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		return
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return
	}

	factory := informers.NewSharedInformerFactory(clientset, 0)

	serviceInformer := factory.Core().V1().Services()
	ingressInformer := factory.Networking().V1().Ingresses()

	controller := pkg.NewController(clientset, serviceInformer, ingressInformer)
	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)
	controller.Run(stopCh)
}