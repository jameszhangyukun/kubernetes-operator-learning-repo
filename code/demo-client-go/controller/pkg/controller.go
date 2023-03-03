package pkg

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	v12 "k8s.io/client-go/informers/core/v1"
	v13 "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	listersnetworkingv1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"reflect"
	"time"
)

const (
	workNum  = 5
	maxRetry = 10
)

type controller struct {
	client        kubernetes.Interface
	ingressLister listersnetworkingv1.IngressLister
	serviceLister listerscorev1.ServiceLister
	queue         workqueue.RateLimitingInterface
}

func NewController(client kubernetes.Interface, serviceInformer v12.ServiceInformer,informer v13.IngressInformer) controller {
	c := controller{
		client:        client,
		ingressLister: informer.Lister(),
		serviceLister: serviceInformer.Lister(),
		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(),"ingressManager"),
	}

	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addService,
		UpdateFunc: c.updateService,
	})
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteIngress,
	})
	return c
}

/**
* 更新service
 */
func (c *controller) updateService(oldObj interface{}, newObj interface{}) {
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	c.enqueue(newObj)
}

func (c *controller) addService(obj interface{}) {
	c.enqueue(obj)
}

func (c *controller) deleteIngress(obj interface{}) {
	ingress, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return
	}
	ownerReference := v1.GetControllerOf(ingress)
	if ownerReference == nil {
		return
	}
	if ownerReference.Kind != "Service" {
		return
	}

	c.queue.Add(ingress.Namespace + "/" + ingress.Name)
}

func (c *controller) Run(stopCh chan struct{}) {
	for i := 0; i < workNum; i++ {
		go wait.Until(c.worker, time.Minute, stopCh)
	}

	<- stopCh
}

func (c *controller) worker() {
	for c.processNextItem() {

	}
}

func (c *controller) processNextItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(item)

	key := item.(string)
	err := c.syncServiceKey(key)
	if err != nil {
		c.handleError(key, err)
	}
	return true
}

func (c *controller) handleError(key string, err error) {
	if c.queue.NumRequeues(key) <= maxRetry {
		c.queue.AddRateLimited(key)
		return
	}

	runtime.HandleError(err)
	c.queue.Forget(key)
}

func (c *controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
	}
	c.queue.Add(key)
}

func (c *controller) syncServiceKey(key string) error {
	namespaceKey, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	service, err := c.serviceLister.Services(namespaceKey).Get(name)
	if errors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	_, ok := service.GetAnnotations()["ingress/http"]
	ingress, err := c.ingressLister.Ingresses(namespaceKey).Get(name)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if ok && errors.IsNotFound(err) {
		newIngress := c.creteIngress(service)
		_, err := c.client.NetworkingV1().Ingresses(namespaceKey).Create(context.TODO(), newIngress, v1.CreateOptions{})
		if err != nil {
			return nil
		}
	} else if !ok && ingress != nil {
		err := c.client.NetworkingV1().Ingresses(namespaceKey).Delete(context.TODO(), name, v1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *controller) creteIngress(service *corev1.Service) *networkingv1.Ingress {
	ingress := networkingv1.Ingress{}

	ingress.ObjectMeta.OwnerReferences = []v1.OwnerReference{
		*v1.NewControllerRef(service, corev1.SchemeGroupVersion.WithKind("Service")),
	}
	ingress.Name = service.Name
	ingress.Namespace = service.Namespace
	pathType := networkingv1.PathTypePrefix
	ingressClassName := "nginx"

	ingress.Spec = networkingv1.IngressSpec{
		IngressClassName: &ingressClassName,

		Rules: []networkingv1.IngressRule{
			{
				Host: "example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathType,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: service.Name,
										Port: networkingv1.ServiceBackendPort{Number: 80},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return &ingress

}
