### Object.runtime
在Kubernetes中Pod、Deployment、Service都是资源，资源像是一种严格定义的，而资源对象对应着具体的数据结构，这些数据结构实现了`Object.runtime`对象
```go
// staging/src/k8s.io/apimachinery/pkg/runtime/interfaces.go

type Object interface {
	GetObjectKind() schema.ObjectKind
	DeepCopyObject() Object
}
```
- DeepCopyObject就是深拷贝对象，将内存中的对象克隆出一个新的对象
- GetObjectKind：就是在处理runtime.Object对象时可以通过GetObjectKind获取对象的具体身份
- 所有的资源对象都是`Object.runtime`的实现

**Unstructed**

```go
type Unstructured struct {
	// Object is a JSON compatible map with string, float, int, bool, []interface{}, or
	// map[string]interface{}
	// children.
	Object map[string]interface{}
}
```
unstructed对象是kubernetes中用来表示资源对象的实体，通常是使用一个map来表示。

使用unstructed实例生成资源对象，是使用`runtime.DefaultUnstructedConvert`的`FromUnstructed`和`ToUnstructed`方法实现。主要的内部实现
```go
// FromUnstructuredWIthValidation converts an object from map[string]interface{} representation into a concrete type.
// It uses encoding/json/Unmarshaler if object implements it or reflection if not.
// It takes a validationDirective that indicates how to behave when it encounters unknown fields.
func (c *unstructuredConverter) FromUnstructuredWithValidation(u map[string]interface{}, obj interface{}, returnUnknownFields bool) error {
    // 获取对象的类型
	t := reflect.TypeOf(obj)
    // 获取对象的类
	value := reflect.ValueOf(obj)
	if t.Kind() != reflect.Ptr || value.IsNil() {
		return fmt.Errorf("FromUnstructured requires a non-nil pointer to an object, got %v", t)
	}

	fromUnstructuredContext := &fromUnstructuredContext{
		returnUnknownFields: returnUnknownFields,
	}
    // 设置具体的值
	err := fromUnstructured(reflect.ValueOf(u), value.Elem(), fromUnstructuredContext)
	if c.mismatchDetection {
		newObj := reflect.New(t.Elem()).Interface()
		newErr := fromUnstructuredViaJSON(u, newObj)
		if (err != nil) != (newErr != nil) {
			klog.Fatalf("FromUnstructured unexpected error for %v: error: %v", u, err)
		}
		if err == nil && !c.comparison.DeepEqual(obj, newObj) {
			klog.Fatalf("FromUnstructured mismatch\nobj1: %#v\nobj2: %#v", obj, newObj)
		}
	}
	if err != nil {
		return err
	}
	if returnUnknownFields && len(fromUnstructuredContext.unknownFieldErrors) > 0 {
		sort.Slice(fromUnstructuredContext.unknownFieldErrors, func(i, j int) bool {
			return fromUnstructuredContext.unknownFieldErrors[i].Error() <
				fromUnstructuredContext.unknownFieldErrors[j].Error()
		})
		return NewStrictDecodingError(fromUnstructuredContext.unknownFieldErrors)
	}
	return nil
}
```

### DynamicClient
对于Deployment、Pod资源，数据结构是固定，对于CRD自定义资源，Clientset就不符合了，所以需要一种Client来操作，所以就是DynamicClient来处理

```go
type dynamicClient struct {
	client *rest.RESTClient
}
```
在DynamicClient中只有一个 RestClient用来操作资源，只有一个关联方法Resource，入参是GVR，另一个结构是`dynamicClient`
```go
func (c *dynamicClient) Resource(resource schema.GroupVersionResource) NamespaceableResourceInterface {
	return &dynamicResourceClient{client: c, resource: resource}
}
```
总结：
1. 和ClientSet不同，DynamicClient提供了统一的操作API，资源需要包装为`Unstructed`数据结构
2. 内部使用RestClient和ApiServer进行交互

### 代码
```go
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

```