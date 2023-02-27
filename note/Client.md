### Client的类型
在client-go中提供了`RESTClient`、`ClientSet`、`DynamicClinet`、`DiscoveryClient`
- RestClient：最基础的客户端，提供了最基本的封装
- Clientset：Client的集合，Clientset中包含了K8s所有内置资源的操作，最常用的客户端
- DynamicClient：动态客户端，可以操作k8s中任意的资源，包括CRD定义的资源
- DiscoveryClient：用于发现k8s提供的资源组、资源版本和资源信息

### RestClient的使用和原理

``` go
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
```

1. 创建config
2. 配置config需要请求的GroupVersion、APIPath、解码协议等
3. 创建RestClient
4. 使用restClient发送请求

原理：在配置中包含ApiServer的所有信息，通过config获取到ApiServer的请求地址、序列化协议等，最后构建HTTP请求发送到ApiServer
```go
// Do formats and executes the request. Returns a Result object for easy response
// processing.
//
// Error type:
//  * If the server responds with a status: *errors.StatusError or *errors.UnexpectedObjectError
//  * http.Client.Do errors are returned directly.
func (r *Request) Do(ctx context.Context) Result {
	var result Result
	err := r.request(ctx, func(req *http.Request, resp *http.Response) {
		result = r.transformResponse(resp, req)
	})
	if err != nil {
		return Result{err: err}
	}
	return result
}

```

### Clientset
```go
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
```
使用client-go来操作Clientset对象来获取资源数据，主要有以下几个步骤：
1. 使用kubeconfig文件或者ServiceAccount来创建访问集群的参数
2. 使用`kubernetes.NewForConfig`来创建Clientset
3. 使用Clientset来操作资源

#### Clientset源码