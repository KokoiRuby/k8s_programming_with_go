### API

[API Server](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/): K8s 控制面的核心组件，集群的前端入口，HTTP Server 暴露 REST API，与 etcd 交互。

所有 API 资源归属于不同组

- core   核心组 `/api`，通过 `/api/v1` 访问。
  - `/api/v1/pods`
  - `/api/v1/namespaces/default/configmaps`
- others 其他组 `/apis`，通过 `/api/{group}/{version}` 访问。
  - `/apis/apps/v1`
  - `/apis/autoscaling/v2`

![img](https://www.redhat.com/rhdc/managed-files/styles/wysiwyg_full_width/private/ohc/API-server-space-1024x604.png.webp?itok=tC9vKcO3)

使用 kubectl 对资源进行 CRUD。

如果多个 Group 存在相同名字资源，则需要通过 `{kind_plural}.{group}` 唯一标识资源

### CRD

通过 CRD 对 K8s API 进行扩展；新增组 `mygroup.com`，HTTP Path `/apis/mygroup.com/{version}`

```yaml
# CRD
cat << EOF | kubectl apply -f -
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  # {kind_plural}.{group} where
  # kind_plural: myresources
  # group: mygroup.com
  name: myresources.mygroup.com
spec:
  group: mygroup.com # group
  names:
    kind: MyResource
    listKind: MyResourceList
    plural: myresources
    singular: myresource
    shortNames: 
    - myres
  # scope
  # Namespaced: allowing multiple instances with the same name across different namespaces.
  # Cluster: each instance is unique across the whole cluster.
  scope: Namespaced 
  versions:
  - name: v1
    served: true  # whether avail for client to interact
    storage: true # whether persisted to etcd, only one version could be true
    schema:
      openAPIV3Schema:
        type: object
        # ++ constraint
        required: ["msg"]
        properties:
          spec:
            type: object
            properties:
              msg:
                type: string
                # ++ constraint
                maxLength: 15
    additionalPrinterColumns: # for kubectl get
    - name: age
      jsonPath: .metadata.creationTimestamp
      type: date
    - name: message
      jsonPath: .spec.msg
      type: string
EOF
```

```yaml
# CR
$ cat > cr-MyResource-test.yaml << EOF
apiVersion: mygroup.com/v1
kind: MyResource
metadata:
  name: test
spec:
  msg: Hello World!
EOF
```

```bash
$ kubectl apply -f cr-MyResource-test.yaml && kubectl get myres
```

### API Discovery

kubelet 通过 API Discovery 机制，查询 `/api` & `/apis` 判断 myres 是否存在属于哪个 group 支持什么操作。

发现 myres 属于 myresources 的简称，之后发起 `/apis/mygroup.com/v1/namespaces/default/myresources`

```bash
$ kubectl get fo --cache-dir $(mktemp -d) -v 6

# Config loaded from file:  /home/eccd/.kube/config
# GET /api?timeout=32s 200 OK in 8 milliseconds
# GET /apis?timeout=32s 200 OK in 2 milliseconds
# GET /apis/mygroup.com/v1/namespaces/default/myresources?limit=500 200 OK in 10 milliseconds
```

```bash
$ kubectl proxy
$ curl -H 'Accept: application/yaml;g=apidiscovery.k8s.io;v=v2beta1;as=APIGroupDiscoveryList' localhost:8001/apis
```

```yaml
apiVersion: apidiscovery.k8s.io/v2beta1
kind: APIGroupDiscoveryList
metadata: {}
items:
# ...
- metadata:
    creationTimestamp: null
    name: mygroup.com
  versions:
  - freshness: Current
    resources:
    - resource: myresources
      responseKind:
        group: mygroup.com
        kind: MyResource
        version: v1
      scope: Namespaced
      shortNames:
      - myres
      singularResource: myresource
      verbs:
      - delete
      - deletecollection
      - get
      - list
      - patch
      - create
      - update
      - watch
    version: v1
```

对于所有 REST API 资源按以下层次发现：

- `GET /api`
- `GET /apis` 返回 APIGroupList，在逐个访问每个 group/version，最终汇聚所有信息
- `GET /apis/{group}`
- `GET /apis/{group}/{version}`

```bash
$ curl -H 'Accept: application/yaml' localhost:8001/api
$ curl -H 'Accept: application/yaml' localhost:8001/apis 
$ curl -H 'Accept: application/yaml' localhost:8001/apis/mygroup.com/v1
```

### kube-apiserver

**Chain of Responsibility 责任链**: `KubeAggregator → KubeAPIServer → ApiExtensionsServer`

```bash
#     │
# kube-aggregator ---> Other APIServers
#     │
#  (delegate)
#     │
# kube-apiserver ---> {core/legacy group /api/**}, {official groups /apis/apps/**, /apis/batch/**, ...}
#     │
#  (delegate)
#     │
#     └── apiextensions-apiserver ---> {CRD groups /apis/apiextensions.k8s.io/**, /apis/<crd.group.io>/**}
#                       │
#                    (delegate)
#                       │
#                       └── notfoundhandler ---> 404 NotFound
```

- **AggregatorServer** 负责处理 apiregistration.k8s.io 组下的资源请求，拦截并转发给 Aggregated APIServer
- **KubeAPIServer** 负责请求通用处理，AuthN/Z，内建资源 CRUD
- **ApiExtensionsServer**
  - 内置 [controllers](https://github.com/kubernetes/apiextensions-apiserver/tree/master/pkg/controller)：
    - **DiscoveryController**
      - 监听 CRD，将 spec 转换为：
        - APIGroupDiscoveryList (1.26+)
        - APIGroup
        - APIResourceList
      - 动态注册 API
        - `/apis/{group}`           → APIGroup 
        - `/apis/{group}/{version}` → APIResourceList
    - **OpenAPIController** v[2/3]
      - 监听 CRD，自动生成并写入 OpenAPI Spec
      - 通过 `/openapi/v[2|3]` 暴露
    - **customresource_handler**
      - 监听 CRD，负责 CR CRUD，持有 RESTStorage (etcd)
      - `/apis/{group}/{version}/{kind_plural}`
      - `/apis/{group}/{version}/namespaces/{namespace}/{kind_plural}`
      - `/apis/{group}/{version}/namespaces/{namespace}/{kind_plural}/{name}`

```bash
$ kubectl proxy
$ curl -s http://localhost:8001/openapi/v3/apis/mygroup.com/v1 | \
jq 'delpaths([path(.), path(..) | select(length >3)])'
```

### [controller-gen](https://github.com/kubernetes-sigs/controller-tools/tree/master/cmd/controller-gen)

Under [controller-tools](https://github.com/kubernetes-sigs/controller-tools) to generate CRD from Go struct. Other Tools: [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)

```bash
$ go install sigs.k8s.io/controller-tools/cmd/controller-gen
$ sudo ln -s $GOPATH/bin/controller-gen /usr/local/bin
```

```bash
$ controller-gen crd:crdVersions=v1 paths=./... output:dir=./artifacts/crd
```

TODO ...