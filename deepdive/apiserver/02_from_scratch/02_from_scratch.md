## APIService

每个 API Group 版本都会有 APIService 与之对应，Local 表示请求在本地进程处理。CRD 创建之后，也会创建对应版本的 APIService。

```bash
$ kubectl get apiservice
$ kubectl get apiservice | awk 'NR==1 || /mygroup/'
```

Delegation

```yaml
apiVersion: apiregistration.k8s.io/v1
kind: APIService
metadata:
  name: v1.mygroup.com
spec:
  group: mygroup.com
  groupPriorityMinimum: 1000
  version: v1
  versionPriority: 100
  # ++ indicate API shall be provided by serivce "apiserver" under hello namespace
  # so that we could replace built-in apiextensions-apiserver by our own
  service:
    name: apiserver
    namespace: hello
  insecureSkipTLSVerify: true
```

```bash
# */apis/mygroup.com/v1/* ---> kube-apiserver ---> mygroup.com-apiserver (√)
#                                     ↓
#                                    (X)
#                                     ↓
#                            apiextensions-apiserver  
```

## Impl

### API Disocvery

`/apis`                → APIGroupList or APIGroupDiscoveryList (since v1.26+) 定义好对象/字符串变量，内存读取直接返回即可。

`/apis/mygroup.com`    → APIGroup 返回 APIGroupList 中的第一个元素

`/apis/mygroup.com/v1` → APIResourceList

```go
mux.Handle("/apis", logHandler(http.HandlerFunc(apis)))

var apiGroupList = metav1.APIGroupList{
	TypeMeta: metav1.TypeMeta{
		Kind:       "APIGroupList",
		APIVersion: "v1",
	},
	// /apis/mygroup.com
	Groups: []metav1.APIGroup{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "APIGroup",
				APIVersion: "v1",
			},
			Name: "mygroup.com",
			Versions: []metav1.GroupVersionForDiscovery{
				{GroupVersion: "mygroup.com/v1", Version: "v1"},
			},
			PreferredVersion: metav1.GroupVersionForDiscovery{
				GroupVersion: "mygroup.com/v1",
				Version:      "v1",
			},
		},
	},
}

func apis(w http.ResponseWriter, r *http.Request) {
	var gvk [3]string

	// 1.27+ kubectl discovery APIGroups and APIResourceList only by /apis with Header
	//    Accept: application/json;g=apidiscovery.k8s.io;v=v2beta1;as=APIGroupDiscoveryList
	// 1.27- kubectl discovery APIGroups and APIResourceList by /apis, /apis/{group}, /apis/{group}/{version}

	// resolve Accept header
	for _, acceptPart := range strings.Split(r.Header.Get("Accept"), ";") {
		// g=apidiscovery.k8s.io      g
		// v=v2beta1                  v
		// as=APIGroupDiscoveryList   k
		if pair := strings.Split(acceptPart, "="); len(pair) == 2 {
			switch pair[0] {
			case "g":
				gvk[0] = pair[1]
			case "v":
				gvk[1] = pair[1]
			case "as":
				gvk[2] = pair[1]
			}
		}
	}

	if gvk[0] == "apidiscovery.k8s.io" && gvk[2] == "APIGroupDiscoveryList" {
		w.Header().Set("Content-Type", "application/json;g=apidiscovery.k8s.io;v=v2beta1;as=APIGroupDiscoveryList")
		_, err := w.Write([]byte(apiGroupDiscoveryList))
		if err != nil {
			return
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		renderJSON(w, apiGroupList)
	}
}
```

```go
mux.Handle("/apis/mygroup.com", logHandler(http.HandlerFunc(apisGroup)))

var apiGroupDiscoveryList = `{
	"apiVersion": "apidiscovery.k8s.io/v2beta1",
	"kind": "APIGroupDiscoveryList",
	"metadata": {},
	"items": [
	  {
		"metadata": {
		  "name": "mygroup.com"
		},
		"versions": [
		  {
			"version": "v1",
			"resources": [
			  {
				"resource": "myresources",
				"responseKind": {
				  "group": "mygroup.com",
				  "kind": "MyResource",
				  "version": "v1"
				},
				"scope": "Namespaced",
				"shortNames": [
				  "myres"
				],
				"singularResource": "myresource",
				"verbs": [
				  "delete",
				  "get",
				  "list",
				  "patch",
				  "create",
				  "update"
				]
			  }
			]
		  }
		]
	  }
	]
  }`

func apisGroup(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	renderJSON(w, apiGroupList.Groups[0])
}


```

```go
mux.Handle("/apis/mygroup.com/v1", logHandler(http.HandlerFunc(apisGroupVersion)))

var apiResourceList = metav1.APIResourceList{
	TypeMeta: metav1.TypeMeta{
		Kind:       "APIResourceList",
		APIVersion: "v1",
	},
	GroupVersion: "mygroup.com/v1",
	APIResources: []metav1.APIResource{
		{
			Name:         "myresources",
			SingularName: "myresource",
			Namespaced:   true,
			Kind:         "MyResource",
			Verbs: []string{
				"create",
				"delete",
				"get",
				"list",
				"update",
				"patch"},
			ShortNames: []string{"myres"},
			Categories: []string{"all"},
		},
	},
}

func apisGroupVersion(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	renderJSON(w, apiResourceList)
}
```

### OpenAPI Spec (Optional)

`/openapi/v2`

`/openapi/v3`

### CRUD

`/apis/mygroup.com/v1/myresources`

`/apis/mygroup.com/v1/namespaces/{namespace}/myresources`

`/apis/mygroup.com/v1/namespaces/{namespace}/myresources/{name}`

## Play