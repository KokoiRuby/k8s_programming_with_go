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

APIs to implement

- API Discovery
  - `/apis`                → APIGroupDiscoveryList
  - `/apis/mygroup.com`    → APIGroup
  - `/apis/mygroup.com/v1` → APIResourceList
- OpenAPI Spec (Optional)
  - `/openapi/v2`
  - `/openapi/v3`
- CRUD
  - `/apis/mygroup.com/v1/myresources`
  - `/apis/mygroup.com/v1/namespaces/{namespace}/myresources`
  - `/apis/mygroup.com/v1/namespaces/{namespace}/myresources/{name}`

### API Disocvery

### OpenAPI Spec

### CRUD

## Play