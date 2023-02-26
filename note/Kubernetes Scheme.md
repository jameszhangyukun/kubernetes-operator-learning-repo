### 介绍
在和apiserver进行通信的时候，需要根据资源对象类型的Group、Version、Kind以及规范定义、编码内容构成Schema类型，然后Clientset对象就可以来访问和操作这些资源类型，Schema的定义主要在api子项目中，源码仓库地址: https://github.com/kubernetes/api ，被同步到 Kubernetes 源码的 staging/src/k8s.io/api 之下。

主要是资源对象的结构体的定义，比如查看`apps/v1`的目录下面的定义：
```
$ tree staging/src/k8s.io/api/apps/v1
staging/src/k8s.io/api/apps/v1
├── BUILD
├── doc.go
├── generated.pb.go
├── generated.proto
├── register.go
├── types.go
├── types_swagger_doc_generated.go
└── zz_generated.deepcopy.go

0 directories, 8 files
```

### types文件
以deploy为例
```go
// Deployment enables declarative updates for Pods and ReplicaSets.
type Deployment struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior of the Deployment.
	// +optional
	Spec DeploymentSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// Most recently observed status of the Deployment.
	// +optional
	Status DeploymentStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}
```
由 TypeMeta、ObjectMeta、DeploymentSpec 以及 DeploymentStatus 4个属性组成，和我们使用 YAML 文件定义的 Deployment 资源对象也是对应的。
```yaml
apiVersion: apps/v1
kind: Deployment  
metadata:
  name:  nginx-deploy
  namespace: default
spec:
  selector:  
    matchLabels:
      app: nginx
  template:  
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: 80
```
其中 apiVersion 与 kind 就是 TypeMeta 属性，metadata 属性就是 ObjectMeta，spec 属性就是 DeploymentSpec，当资源部署过后也会包含一个 status 的属性，也就是 DeploymentStatus ，这样就完整的描述了一个资源对象的模型。

#### zz_generated.deepcopy.go 文件
这个文件是由deepcopy-gen工具创建的定义资源类型的`DeepCopyObject()`方法的文件，所有注册到Schema的资源类型都要实现`runtime.Object`接口
```go
// staging/src/k8s.io/apimachinery/pkg/runtime/interface.go
type Object interface {
	GetObjectKind() schema.ObjectKind
	DeepCopyObject() Object
}
```
而所有的资源类型都包含一个 TypeMeta 类型，而该类型实现了 GetObjectKind() 方法，所以各资源类型只需要实现 DeepCopyObject() 方法即可：
```go
// staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/meta.go
func (obj *TypeMeta) GetObjectKind() schema.ObjectKind { return obj }
```
各个资源类型的 DeepCopyObject() 方法也不是手动定义，而是使用 deepcopy-gen 工具命令统一自动生成的，该工具会读取 types.go 文件中的 +k8s:deepcopy-gen 注释。

### register.go文件
`register.go` 文件的主要作用是定义了AddToScheme函数，将各种资源注册到Clientset使用的Schema对象中
```go
// staging/src/k8s.io/api/apps/v1/register.go
var (
	// TODO: move SchemeBuilder with zz_generated.deepcopy.go to k8s.io/api.
	// localSchemeBuilder and AddToScheme will stay in k8s.io/kubernetes.
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
	localSchemeBuilder = &SchemeBuilder
	// 对外暴露的 AddToScheme 方法用于注册该 Group/Verion 下的所有资源类型
	AddToScheme        = localSchemeBuilder.AddToScheme
)

// staging/src/k8s.io/client-go/kubernetes/scheme/register.go
// 新建一个 Scheme，将各类资源对象都添加到该 Scheme
var Scheme = runtime.NewScheme()  
// 为 Scheme 中的所有类型创建一个编解码工厂
var Codecs = serializer.NewCodecFactory(Scheme)
// 为 Scheme 中的所有类型创建一个参数编解码工厂
var ParameterCodec = runtime.NewParameterCodec(Scheme)
// 将各 k8s.io/api/<Group>/<Version> 目录下资源类型的 AddToScheme() 方法注册到 SchemeBuilder 中
var localSchemeBuilder = runtime.SchemeBuilder{
	......
	appsv1.AddToScheme, 
	appsv1beta1.AddToScheme,
	appsv1beta2.AddToScheme,
	......
}

var AddToScheme = localSchemeBuilder.AddToScheme

func init() {
	v1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
	// 调用 SchemeBuilder 中各资源对象的 AddToScheme() 方法，将它们注册到到 Scheme 对象
	utilruntime.Must(AddToScheme(Scheme))
}
```
将各类资源类型注册到`**全局的 Scheme 对象**`中，这样 Clientset 就可以识别和使用它们了。

apimachinery 子项目主要是 Kubernetes 服务端和客户端项目都共同依赖的一些公共方法、struct、工具类的定义，主要服务于 kubernetes、client-go、apiserver 这三个项目。