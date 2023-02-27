DiscoveryClient是面向资源的，例如查看当前Kubernetes有哪些Group、Version、Resource。
```go
type DiscoveryClient struct {
	restClient restclient.Interface

	LegacyPrefix string
}
```
其中包含RestClient和LegacyPrefix，LegacyPrefix就是请求资源Url中的前缀
### 代码
```go
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
	// 先看Group信息
	fmt.Printf("APIGroup :\n\n %v\n\n\n\n", apiGroup)
	// APIResourceListSlice是个切片，里面的每个元素代表一个GroupVersion及其资源
	for _, singleAPIResourceList := range apiResourceList {
		// GroupVersion是个字符串，例如"apps/v1"
		groupVerionStr := singleAPIResourceList.GroupVersion
		// ParseGroupVersion方法将字符串转成数据结构
		gv, err := schema.ParseGroupVersion(groupVerionStr)
		if err != nil {
			panic(err)
		}
		fmt.Println("*****************************************************************")
		fmt.Printf("GV string [%v]\nGV struct [%#v]\nresources :\n\n", groupVerionStr, gv)
		// APIResources字段是个切片，里面是当前GroupVersion下的所有资源
		for _, singleAPIResource := range singleAPIResourceList.APIResources {
			fmt.Printf("%v\n", singleAPIResource.Name)
		}
	}
}

```