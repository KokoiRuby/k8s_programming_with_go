[toc]

## [API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)

简化客户端开发，保证针对一系列用例的配置原理实现的一致性。

K8s API 基于 RESTful 风格，通过标准 HTTP verb 对一个对象的描述进行 CRUD，以 JSON 格式进行交互。

### Types/Kinds

**Types/Kinds** (top-level) the name of a particular object **schema**, grouped into three categories:

1. **Objects** 表示系统中**持久化**实体 `Pod`, `Service`, `Node`
2. **List** 表示 Resource 集合 `PodList`, `ServiceList`
3. **Simple** 表示 objects 上的具体操作，**非持久化**实体。典型 subresources：`/binding`, `/status`, `/scale`

### Resources

**Resource** 系统实体的表现形式, **sent/recv as JSON (obj)** via HTTP to the server。

Examples: kind "Pod" is exposed as a "pods" resource.

所有由 API 返回的 JSON 对象必须包含以下字段：

- kind 标识该对象所属的 schema
- apiVersion 标识该对象所属的 schema 的版本

![image20240815080746148](./02_api_conventions.assets/image-20240815080746148.png)

### Objects

#### Metadata

每一个对象 metadata 都**必须**包含 namespace/name/uid。

每一个对象 metadata 都**应该**包含 resourceVersion/generation/creationTimestamp/deletionTimestamp/labels/annotations。

#### Spec & Status

Spec 描述对象的期望状态（通常可读写）。Status 描述当前/最近观察到的对象状态（通常只读），必须提供 /status subresource。

System‘s behavior is **level-based** rather than edge-based.

例：如果一个值从 2 修改到 5 之后修改到 3，没有必要先到 5 再到 3，而是直接到 3 即可。

对于一些非持久化的对象，比如 `SubjectAccessReview` 以及其他 webhook 调用，Spec & Status 可以用于封装请求/响应。

**Status 属性**

**[Conditions](https://github.com/kubernetes/apimachinery/blob/release-1.23/pkg/apis/meta/v1/types.go#L1432-L1492)** 为 controller 提供 high-level 状态报告（类摘要），而无需了解其细节。

一个对象可以由多次状态报告，因此 condition 是一个切片。

#### References to related objects

#### Lists of named subobjects preferred over maps

```yaml
# list, preferred
ports:
  - name: www
    containerPort: 80

# map, nok
ports:
  www:
    containerPort: 80
```

#### [Primitive types](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#primitive-types)

#### [Constants](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#primitive-types)

CamelCase string instead.

### Lists and Simple kinds

每一个对象 metadata 都**应该**包含：resourceVersion

对于 simple kind 确保支持幂等和乐观锁控制，∵ 它们通常是一些 action 参考的依据。

### Differing Representations

一个 API 对应的实体对于不同的客户端，可以有不同的表现，或者在系统出现某个转变时，会转换成另一个对象。

例：Service 可以表现为 2 种不同的资源：Service 自身以及 Endpoints。

### Verbs on Resources

`GET /<resourceNamePlural>`

`POST /<resourceNamePlural>`

`GET /<resourceNamePlural>/<name>`

`DELETE /<resourceNamePlural>/<name>`

`DELETE /<resourceNamePlural>`

`PUT /<resourceNamePlural>/<name>`

`PATCH /<resourceNamePlural>/<name>`

- JSON Patch
  - `Content-Type: application/json-patch+json`
  - a sequence of operations that are executed on the resource
  - 可以对文档中的任意位置进行操作
  - 当需要对资源的某些部分进行精确的增删改操作时，JSON Patch 是最合适的
- Merge Patch
  - `Content-Type: application/merge-patch+json`
  - The submitted JSON is "merged" with the current resource to create a new one, then the new one is saved.
  - 无法删除字段，只能更新或添加字段
  - 适用于简单地更新现有字段的场景，不适合复杂的文档操作
- Strategic Merge Patch 
  - `Content-Type: application/strategic-merge-patch+json`
  - 够对复杂对象的部分字段进行更新，并支持对列表项的精确操作
  - 适合对 K8s 资源的复杂更新操作，比如更新修改容器列表中断额某个容器

`GET /<resourceNamePlural>?watch=true`

### Short-names and Categories

对于内建资源，由 REST handler `ShortNames() []string` & `Categories() []string` 实现。

对于自定义资源，由 `.spec.names.shortNames` and `.spec.names.categories` 定义。

#### Short-names

As **hints** to resolve ambiguous user invocations to a single resource.

```bash
$ kubectl get sts
$ kubectl get statefulsets.v1.apps
```

#### Categories

As **hints** to resolve user invocations to multiple resources.

```bash
$ kubectl get all
$ kubectl get pods.v1.,\
                replicationcontrollers.v1.,\
                services.v1.,\
                daemonsets.v1.apps,deployments.v1.apps,\
                replicasets.v1.apps,statefulsets.v1.apps,\
                horizontalpodautoscalers.v2.autoscaling,\
                cronjobs.v1.batch,jobs.v1.batch
```

### Idempotency

所有兼容 API 必须支持“命名幂等性”，当一个 POST 请求作用于同名对象时，需返回 409 冲突。

`metadata.generateName` 确保持久化前唯一。

### Optional vs. Required

Optional

- `+optional` comment tag
- `omitempty` json tag 进行 json 编码时，空值会忽略
- **pointer type** 选择指针类型原因：
  - 区分未设置和零值 nil vs. 0 (int)
  - 保持后向兼容
  - 支持显式删除
  -  `omitempty` tag 用于跳过零值字段的序列化。指针类型可以更自然地与 JSON 的 `null` 表现形式对应
- POST/PUT 支持未设置可选字段

Required

- no `+optional` comment tag
- no `omitempty` json tag
- not pointer type
- POST/PUT 不支持未设置必须字段

### Defaulting

应用默认值方式有

1. Static
   - specific to each API ver
   - may be conditional on or deterministically derived from other fields
   - defaulting logic is unaware of updates and has no concept of "previous value"
2. Admission Control
   - 默认值不由相关对象派生，比如 StorageClass & PVC
3. Controllers/Late
   - 默认值不由相关对象派生，`pod.spec.nodeName`

#### What May Be Defaulted

- 设置先前未设置的字段
- 向 map 中添加 keys
- 向可 merge 的数组中添加值

当我们永远不像改变值或者常数覆盖用户提供的值，若请求了无效的内容，应该返回一个错误。

#### Considerations For PUT Operations

一旦对象创建赋予了默认值，随着时间的推移，时常会对其进行更新。

PUT 原语要求幂等，但在更新的场景下并不是很合适。

当对象通过 PUT 更新时，API Server 将看到具有先前分配的默认值的“旧”对象和具有新分配的默认值的“新”对象。

### Concurrency Control and Consistency

基于 ResourceVersion 的乐观锁控制，标识对象的内部版本，每次修改对象，都会更新；409 conflict if not matched

客户端获取 resourceVersion 预期值的唯一方法只有从 API Server 进行 GET，必须原封不动返回给 API Server 进行冲突判断。

可通过 Watch 从指定 resourceVersion 开始监视，确保不会错过 GET 之后的任何变化。

### Serialization Format

API 响应的输入和输出的默认序列化必须是 JSON。

内置资源接受 protobuf 编码。

日期 RFC3339 字符串。

### Units

单位必须在字段名称中明确显示 timeSeconds，或者必须指定为值的一部分 resource.Quantity。

Duration 字段必须表示为整数字段，单位是字段名称的一部分 leaseDurationSeconds。

### Selecting Fields

建议使用标准 JavaScript 语法来访问该字段 `.` 操作符 `metadata.name` 或 `fields[1].state.current`

### Object references

命名空间类型上的对象引用通常应仅引用同一命名空间中的对象，不要跨命名空间引用。

#### Naming of the reference field

参考字段的名称应采用 `{field}Ref` 格式。`targetRef` in an endpoint.

#### Referencing resources with multiple versions

**Controller**应该假设资源的版本可能会改变，并包括适当的错误处理。

#### Handling of resources that do not exist

资源不存在：

- 版本不存在
- 集群引导过程中竞争导致资源未添加
- 用户错误

**Controller** 应在引用资源可能不存在的假设下编写，并包含错误处理以便让用户清楚地了解问题。

#### Validation of fields

使用 Controller 在将字段用作 API 请求中的路径段之前验证字段，并发出事件以告知用户验证失败。

#### Do not modify the referred object

不要修改所引用的对象，或者将修改限制在同一命名空间中的对象并限制允许的修改类型 HorizontalPodAutoscaler 仅写入 /scale 子资源。

#### Minimize copying or printing values to the referrer object

一般来说，不要将从引用对象检索到的信息写入或打印到规范、其他对象或日志中。

Controller 的权限可能与控制者所管理对象的作者的权限不同，因此对象的作者可能没有查看引用对象的权限。

#### Example

**单一资源对象引用**很简单，因为控制器可以对识别对象所需的大多数限定符进行硬编码。

控制器行为：需要从中检索值的对象的 GVR，并且可以使用 DiscoveryClient 或直接构建 API 路径。

```yaml
# for a single resource, the suffix should be Ref, with the field name
# providing an indication as to the resource type referenced.
secretRef:
    name: foo
    # namespace would generally not be needed and is discouraged,
    # as explained above.
    namespace: foo-namespace
```

**多资源对象引用**（可指向一组有限的有效资源类型），同单一资源对象引用，操作员可以提供缺失的字段

```yaml
# guidance for the field name is the same as a single resource.
fooRef:
    group: sns.services.k8s.aws
    resource: topics
    name: foo
    namespace: foo-namespace
```

**通用对象引用**，当需要提供指向某个对象的**指针**以简化用户的发现时。

控制器行为：通过 DiscoveryClient 找到资源，由于任何可检索字段对于所有对象都是通用的，因此资源的任何版本都应该可以。

```yaml
fooObjectRef:
    group: operator.openshift.io
    resource: openshiftapiservers
    name: cluster
    # namespace is unset if the resource is cluster-scoped, or lives in the
    # same namespace as the referrer.
```

**字段引用**，当需要从引用对象中的特定字段提取值时，需要显式版本。

控制器行为：用户将提供所有必需的路径元素：组、版本、资源、名称以及可能的命名空间 `/apis/{group}/{version}/{resource}/`

```yaml
fooFieldRef:
   version: v1 # version of the resource
   # group is elided in the ConfigMap example, since it has a blank group in the OpenAPI spec.
   resource: configmaps
   fieldPath: data.foo
```

### [HTTP Status codes](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#http-status-codes)

### [Response Status Kind](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#response-status-kind)

发生错误时，K8s 将始终从任何 API 端点返回 `Status` 类型。客户端应在适当的时候处理这些类型的对象。

- 当操作不成功，即非 2xx HTTP status code
- HTTP Delete 调用成功时

```bash
$ curl -v -k -H "Authorization: Bearer WhCDvq4VPpYhrcfmF6ei7V9qlbqTubUc" https://10.240.122.184:443/api/v1/namespaces/default/pods/grafana

> GET /api/v1/namespaces/default/pods/grafana HTTP/1.1
> User-Agent: curl/7.26.0
> Host: 10.240.122.184
> Accept: */*
> Authorization: Bearer WhCDvq4VPpYhrcfmF6ei7V9qlbqTubUc
>

< HTTP/1.1 404 Not Found
< Content-Type: application/json
< Date: Wed, 20 May 2015 18:10:42 GMT
< Content-Length: 232
<
{
  "kind": "Status",
  "apiVersion": "v1",
  "metadata": {},
  "status": "Failure",                    # either Success or Failure
  "message": "pods \"grafana\" not found",# human-readable description of the error
  "reason": "NotFound",                   # machine-readable, one-word, CamelCase description of why Failure
  "details": {                            # extended data associated with the reason
    "name": "grafana",
    "kind": "pods"
  },
  "code": 404
}
```

### Events

Events 是状态信息的补充，因为除了当前或以前的状态外，它们还可以提供有关状态和发生事件的一些历史信息。

A unique, specific, short, CamelCase reason for each event category

### [Naming conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#naming-conventions)

### Label, selector, and annotation conventions

Label 属于用户领域。它们旨在使用对用户有意义（而不是对系统有意义的）的属性来促进 API 资源的组织和管理。

Annotation 的预期用途与标签大不相同。它们主要由工具和系统扩展生成和使用，或由最终用户用来参与组件的非标准行为。可用管理域作 Key。

`kubernetes.io` 和 `k8s.io` 域下的键前缀是为 kubernetes 项目保留的，第三方不得使用。

### [WebSockets & SPDY](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#websockets-and-spdy)

### Validation

API 服务器收到 API 对象后，会对其进行验证。验证失败返回 `Failure` status with `reason` set to `Invalid` 给调用者。

[Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#validation)

### [Representing Allocated Values](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#representing-allocated-values)

许多 API 类型都包含代表用户从某个较大空间分配的值，这些分配通常由控制器以异步方式驱动，与用户的 API 操作无关。

系统不应该信任具有此类字段的用户，并且必须在使用之前验证或以其他方式确认此类请求。

#### When to use a spec filed

#### When to use a spec filed

#### When to use a different type