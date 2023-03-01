package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"time"
)

func main()  {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err !=nil{
		panic(err)
		return
	}
	// 创建Clientset对象
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
		return
	}
	informerFactory := informers.NewSharedInformerFactory(clientset, 30*time.Second)
	deploymentsInformer := informerFactory.Apps().V1().Deployments()
	informer := deploymentsInformer.Informer()
	lister := deploymentsInformer.Lister()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc  :  func(obj interface{}){},
		UpdateFunc: func(oldObj, newObj interface{}){},
		DeleteFunc :func(obj interface{}){},
	})
	stopChan := make(chan struct{})
	defer close(stopChan)

	// 启动所有的Informer List & watch
	informerFactory.Start(stopChan)
	// 等待所有的Informer的缓存被同步
	informerFactory.WaitForCacheSync(stopChan)

	deployments, err := lister.Deployments("kube-system").List(labels.Everything())
	if err != nil {
		panic(err)
		return
	}
	for _,deployment := range deployments {
		fmt.Println(deployment.Name)
	}
}
