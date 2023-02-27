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
1. 创建流程

```go
// staging/src/k8s.io/client-go/kubernetes/clientset.go

// NewForConfig 使用给定的 config 创建一个新的 Clientset
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}
	var cs Clientset
	var err error
	cs.admissionregistrationV1beta1, err = admissionregistrationv1beta1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	// 将其他 Group 和版本的资源的 RESTClient 封装到全局的 Clientset 对象中
	cs.appsV1, err = appsv1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	......
	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	return &cs, nil
}
```
在`NewForConfig`中将各种资源的RESTClient封装到Clientset中，需要访问某个资源时获取对应的Client即可。主要的初始化流程就是，在`NewForConfigAndClinet()`中
```go
// 创建对应资源的Client
func NewForConfigAndClient(c *rest.Config, h *http.Client) (*AppsV1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientForConfigAndClient(&config, h)
	if err != nil {
		return nil, err
	}
	return &AppsV1Client{client}, nil
}
// 这里配置对应资源的Client
func setConfigDefaults(config *rest.Config) error {
	gv := v1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}
```
List资源的源码
```go
// staging/src/k8s.io/client-go/kubernetes/typed/apps/v1/deployment.go

func (c *deployments) List(opts metav1.ListOptions) (result *v1.DeploymentList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.DeploymentList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("deployments").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}
```
其实就是调用创建RestClient向对应的资源发送请求。实际发送网路请求的代码
```go
// staging/src/k8s.io/client-go/rest/request.go
// request connects to the server and invokes the provided function when a server response is
// received. It handles retry behavior and up front validation of requests. It will invoke
// fn at most once. It will return an error if a problem occurred prior to connecting to the
// server - the provided function is responsible for handling server errors.
func (r *Request) request(ctx context.Context, fn func(*http.Request, *http.Response)) error {
	//Metrics for total request latency
	start := time.Now()
	defer func() {
		metrics.RequestLatency.Observe(r.verb, r.finalURLTemplate(), time.Since(start))
	}()

	if r.err != nil {
		klog.V(4).Infof("Error in request: %v", r.err)
		return r.err
	}

	if err := r.requestPreflightCheck(); err != nil {
		return err
	}
	// 初始化网络客户端
	client := r.c.Client
	if client == nil {
		client = http.DefaultClient
	}

	// Throttle the first try before setting up the timeout configured on the
	// client. We don't want a throttled client to return timeouts to callers
	// before it makes a single request.
	if err := r.tryThrottle(ctx); err != nil {
		return err
	}
	// 超时处理
	if r.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.timeout)
		defer cancel()
	}

	// 重试机制，最多重试10次
	maxRetries := 10
	retries := 0
	for {

		url := r.URL().String()
		// 构造请求对象
		req, err := http.NewRequest(r.verb, url, r.body)
		if err != nil {
			return err
		}
		req = req.WithContext(ctx)
		req.Header = r.headers

		r.backoff.Sleep(r.backoff.CalculateBackoff(r.URL()))
		if retries > 0 {
			// We are retrying the request that we already send to apiserver
			// at least once before.
			// This request should also be throttled with the client-internal rate limiter.
			if err := r.tryThrottle(ctx); err != nil {
				return err
			}
		}
		// 发起网络调用
		resp, err := client.Do(req)
		updateURLMetrics(r, resp, err)
		if err != nil {
			r.backoff.UpdateBackoff(r.URL(), err, 0)
		} else {
			r.backoff.UpdateBackoff(r.URL(), err, resp.StatusCode)
		}
		if err != nil {
			// "Connection reset by peer" or "apiserver is shutting down" are usually a transient errors.
			// Thus in case of "GET" operations, we simply retry it.
			// We are not automatically retrying "write" operations, as
			// they are not idempotent.
			if r.verb != "GET" {
				return err
			}
			// For connection errors and apiserver shutdown errors retry.
			if net.IsConnectionReset(err) || net.IsProbableEOF(err) {
				// For the purpose of retry, we set the artificial "retry-after" response.
				// TODO: Should we clean the original response if it exists?
				resp = &http.Response{
					StatusCode: http.StatusInternalServerError,
					Header:     http.Header{"Retry-After": []string{"1"}},
					Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
				}
			} else {
				return err
			}
		}

		done := func() bool {
			// Ensure the response body is fully read and closed
			// before we reconnect, so that we reuse the same TCP
			// connection.
			defer func() {
				const maxBodySlurpSize = 2 << 10
				if resp.ContentLength <= maxBodySlurpSize {
					io.Copy(ioutil.Discard, &io.LimitedReader{R: resp.Body, N: maxBodySlurpSize})
				}
				resp.Body.Close()
			}()

			retries++
			if seconds, wait := checkWait(resp); wait && retries < maxRetries {
				if seeker, ok := r.body.(io.Seeker); ok && r.body != nil {
					_, err := seeker.Seek(0, 0)
					if err != nil {
						klog.V(4).Infof("Could not retry request, can't Seek() back to beginning of body for %T", r.body)
						fn(req, resp)
						return true
					}
				}

				klog.V(4).Infof("Got a Retry-After %ds response for attempt %d to %v", seconds, retries, url)
				r.backoff.Sleep(time.Duration(seconds) * time.Second)
				return false
			}
			// 将req和resp回调
			fn(req, resp)
			return true
		}()
		if done {
			return nil
		}
	}
}
```