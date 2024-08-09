[client-go](https://github.com/kubernetes/client-go) a high-level library for talking to K8s cluster.

It **merges** K8s API & API Machinery lib.

It also provides a set of clients to **execute op** on the resources of the K8s API.

```go
import (
	"k8s.io/client-go/kubernetes"
)
```

```bash
# align with K8s
$ go get k8s.io/client-go@vX.Y.Z
```

## Connecting to the Cluster

`rest.Config` in `rest` package contains all conf info necessary for an app to a REST API Server.

**In-cluster Conf**

- **sa** token & root cert under `/var/run/secrets/kubernetes.io/serviceaccount/`
  - `automountServiceAccountToken: false`
- **env**: `KUBERNETES_SERVICE_HOST:KUBERNETES_SERVICE_PORT` added by kubelet

```go
import "k8s.io/client-go/rest"
func InClusterConfig() (*Config, error)
```

Out-of-cluster Conf**

- **kubeconfig** from mem []byte.

```go
import "k8s.io/client-go/tools/clientcmd"
 
configBytes, err := os.ReadFile(
	"/home/user/.kube/config",
)
if err != nil {
	return err
}

config, err := clientcmd.RESTConfigFromKubeConfig(configBytes,)
if err != nil {
	return err
}
```

- **kubeconfig** from disk.

```go
import "k8s.io/client-go/tools/clientcmd"

config, err := clientcmd.BuildConfigFromFlags(
    "",
    "/home/user/.kube/config",
)

// override URL of API Server
config, err := clientcmd.BuildConfigFromFlags(
    "https://192.168.1.10:6443",
    "/home/user/.kube/config",
)
```

- **kubeconfig** personalized, `BuildConfigFromKubeconfigGetter` function accepting a `kubeconfigGetter` function as an argument, which itself will return an `api.Config` structure.

```go
import (
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/tools/clientcmd/api"
)

config, err := clientcmd.BuildConfigFromKubeconfigGetter(
    "",
    func() (*api.Config, error) {
        apiConfig, err := clientcmd.LoadFromFile("/home/user/.kube/config")
        if err != nil {
            return nil, err
        }
        // TODO: manipulate apiConfig
        return apiConfig, nil
    },
)

```

- **kubeconfig** specify another & merge several.

```go
import (
    "k8s.io/client-go/tools/clientcmd"
)

config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
    clientcmd.NewDefaultClientConfigLoadingRules(),
    nil,
).ClientConfig()

```

- kubeconfig override CLI flags. [3pp](https://github.com/spf13/pflag).

```go
import (
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"
)
var (
	flags pflag.FlagSet
	overrides clientcmd.ConfigOverrides
	of = clientcmd.RecommendedConfigOverrideFlags("")
)
clientcmd.BindOverrideFlags(&overrides, &flags, of)
flags.Parse(os.Args[1:])
config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
	clientcmd.NewDefaultClientConfigLoadingRules(),
	&overrides,
).ClientConfig()
```

## Clientset

Clientset is a set of clients, and **each client is dedicated** to its own Group/Version.

Create client by `*rest.Config`

```go
// default HTTP client built
func NewForConfig(c *rest.Config) (*Clientset, error)
func NewForConfigOrDie(c *rest.Config) *Clientset     // panic if err

// personaliz HTTP client
func NewForConfigAndClient(c *rest.Config, httpClient *http.Client) (*Clientset, error)
```

Interface

```go
type Interface interface {
    // discover GVR avail in cluster.
    Discovery() discovery.DiscoveryInterface
    [...]
    AppsV1() appsv1.AppsV1Interface
    AppsV1beta1() appsv1beta1.AppsV1beta1Interface
    AppsV1beta2() appsv1beta2.AppsV1beta2Interface
    [...]
    CoreV1() corev1.CoreV1Interface
    [...]
}
```

Each method returns a value that implements an interface specific to the G/V.

```go
type CoreV1Interface interface {
    // get a REST client for the specific G/V
    RESTClient() rest.Interface
    ComponentStatusesGetter
    ConfigMapsGetter
    EndpointsGetter
    [...]
}

// rest.Interface implements a series of methods & return a Request obj with Verb
type Interface interface {
    GetRateLimiter() flowcontrol.RateLimiter
    Verb(verb string) *Request
    Post() *Request
    Put() *Request
    Patch(pt types.PatchType) *Request
    Get() *Request
    Delete() *Request
    APIVersion() schema.GroupVersion
}

type ConfigMapsGetter interface {
	ConfigMaps(namespace string) ConfigMapInterface
}

// Each method related to an operation takes as a parameter an Option structure
type ConfigMapInterface interface {
    Create(
    	ctx context.Context,
    	configMap *v1.ConfigMap,
    	opts metav1.CreateOptions,
    ) (*v1.ConfigMap, error)
    Update(
    	ctx context.Context,
    	configMap *v1.ConfigMap,
    	opts metav1.UpdateOptions,
    ) (*v1.ConfigMap, error)
    Delete(
    	ctx context.Context,
    	name string,
    	opts metav1.DeleteOptions,
    ) error
    [...]
}
```

Use **chaining** to make an operation on a resource of a GV.

```go
clientset.
	GroupVersion().
	NamespacedResource(namespace).
	Operation(ctx, options)

clientset.
	GroupVersion().
	NonNamespacedResource().	
	Operation(ctx, options)
```

```go
// list the Pods of the core/v1 Group/Version in namespace default
podList, err := clientset.
    CoreV1().
    Pods("default").
    List(ctx, metav1.ListOptions{})

// list of pods in all namespaces
podList, err := clientset.
    CoreV1().
    Pods("").
    List(ctx, metav1.ListOptions{})

// list of nodes
nodesList, err := clientset.
    CoreV1().
    Nodes().
    List(ctx, metav1.ListOptions{})
```

## Examining the Requests

[klog](https://github.com/kubernetes/klog) forked from glog.

```go
import (
	"flag"
    "k8s.io/klog/v2"
)

func main() {
    klog.InitFlags(nil)  // 6+ to get URL ffor each request
    flag.Parse()
    [...]
}
```

## Creating a Resource

Declare first using dedicated **Kind** struct, then use `Create` method.

```go
// declare pod strcut
wantedPod := corev1.Pod{
    Spec: corev1.PodSpec{
        Containers: []corev1.Container{
            {
                Name:  "nginx",
                Image: "nginx",
            },
        },
    },
}
wantedPod.SetName("nginx-pod")

// create pod
createdPod, err := clientset.
						CoreV1().
						Pods("project1").
						Create(ctx, &wantedPod, v1.CreateOptions{})
```

Avail `CreateOptions`

- **DryRun**
- **FieldManager**: for server-side apply
- **FieldValidation**: how the server should react when duplicate or unknown fields.
  - `metav1.FieldValidationIgnore`
  - `metav1.FieldValidationWarn`
  - `metav1.FieldValidationStrict`

Possible errors in `k8s.io/apimachinery/pkg/api/errors`

- `IsAlreadyExists`
- `IsNotFound`
- `IsInvalid`

## Getting Info About a Resource

```go
pod, err := clientset.
                CoreV1().
                Pods("default").
                Get(ctx, "nginx-pod", metav1.GetOptions{})
```

Avail `GetOptions`

- ResourceVersion：latest if 0 but not guaranteed.

Possible errors in `k8s.io/apimachinery/pkg/api/errors`

- `IsNotFound`

## Getting List of Resources

```go
podList, err := clientset.
                    CoreV1().
                    Pods("default").
                    List(ctx, metav1.ListOptions{})

podList, err := clientset.
                    CoreV1().
                    Pods("").
                    List(ctx, metav1.ListOptions{})
```

Avail `ListOptions`

- **LabelSelector, FieldSelector**
- **Watch, AllowWatchBookmarks**
- **ResourceVersion**: latest if 0 but not guaranteed.
- **ResourceVersionMatch**：`metav1.ResourceVersionMatchExact`
- **TimeoutSeconds**
- **Limit, Continue**

Possible errors in `k8s.io/apimachinery/pkg/api/errors`

- `IsResourceExpired`

## Filtering the Result of a List

### LabelSelector

`label` pkg & `Requirement` obj.

```go
import (
	"k8s.io/apimachinery/pkg/labels"
)

func NewRequirement(
	key string,
	op selection.Operator,
    vals []string,
	opts ...field.PathOption,
) (*Requirement, error)
```

`selection.Operator` in `k8s.io/apimachinery/pkg/selection`

- **selection.In/NotIn**
- **selection.Equals/DoubleEquals/NotEquals**
- **selection.Exists/DoesNotExist**
- **selection.Gt/Lt**

```go
req1, err := labels.NewRequirement(
    "myKey",
    selection.Equals,
    []string{"myVal"},
)
```

Add `Requirement` to selector using `Add` then to string.

```go
labelsSelector = labelsSelector.Add(*req1, *req2)
s := labelsSelector.String()
```

**Parse from string** 

```go
selector, err := labels.Parse(
	"mykey = value1, count < 5",
)
if err != nil {
	return err
}
s := selector.String()
```

**Parse from K/V**

```go
set := labels.Set{
    "key1": "value1",
	"key2": "value2",
}

selector, err = labels.ValidatedSelectorFromSet(set)
s = selector.String()
```

### FieldSelector

`fields` pkg.

```go
import (
	"k8s.io/apimachinery/pkg/fields"
)

func OneTermEqualSelector(k, v string) Selector
func OneTermNotEqualSelector(k, v string) Selector
func AndSelectors(selectors ...Selector) Selector
```

```go
fselector = fields.AndSelectors(
    fields.OneTermEqualSelector(
    	"status.Phase",
    	"Running",
    ),
    fields.OneTermNotEqualSelector(
    	"spec.restartPolicy",
    	"Always",
    ),
)
```

**Parse from string**

```go
selector, err := fields.ParseSelector(
	"status.Phase=Running, spec.restartPolicy!=Always",
)
if err != nil {
	return err
}
s := selector.String()
```

**Parse from K/V**

```go
set := fields.Set{
    "field1": "value1",
    "field2": "value2",
}
selector = fields.SelectorFromSet(set)
s = selector.String()
```

## Deleting a Resource

The Delete operation will not effectively delete the resource, but mark the resource to be deleted `.metadata.deletionTimestamp`, and the deletion will happen **asynchronously**.

```go
err = clientset.
        CoreV1().
        Pods("default").
        Delete(ctx, "nginx-pod", metav1.DeleteOptions{})
```

Avail `DeleteOptions`

- **Dry**Run
- **GracePeriodSeconds**: val must be a pointer to a non-negative integer.
- **Preconditions**: indicate which resource you expect to delete.
  - UID
  - ResourceVersion
- **PropagationPolicy**: indicates whether and how GC will be performed in terms of "OwnerReferences”
  - `metav1.DeletePropagationOrphan`: orphanize & not delted by GC.
  - `metav1.DeletePropagationBackground`: non-blocking.
  - `metav1.DeletePropagationForeground`: blocking.

Possible errors in `k8s.io/apimachinery/pkg/api/errors`

- `IsNotFound`
- `IsConflict`

## Deleting a Collection of Resources

```go
err = clientset.
    	CoreV1().
    	Pods("default").
    	DeleteCollection(
    		ctx,
    		metav1.DeleteOptions{},  // Deleting a Resource
			metav1.ListOptions{},    // Getting List of Resources
		)
```

## Updating a Resource

```go
updatedDep, err := clientset.
                        AppsV1().
                        Deployments("default").
                        Update(
                            ctx,
                            myDep,
                            metav1.UpdateOptions{}, // Creating a Resource
                        )
```

Possible errors in `k8s.io/apimachinery/pkg/api/errors`

- `IsInvalid`
- `IsConflict`

## Strategic Merge Patch

Only parts that u want to modify, types:

- MergePatchType 简单的基于 JSON 的 patch。
- StrategicMergePatchType 提供了一种更复杂、更精细的方式来应用对资源的修改，保留未提及的字段。

```go
Patch(
    ctx context.Context,
    name string,
    pt types.PatchType,    // StrategicMergePatchType or MergePatchType
    data []byte,
    opts metav1.PatchOptions,
    subresources ...string,
) (result *v1.Deployment, err error)

```

```go
patch := client.StrategicMergeFrom(
	createdDep,                              // obj u want to patch
	pkgclient.MergeFromWithOptimisticLock{}, // add ResourceVersion to patch
)

updatedDep := createdDep.DeepCopy()          // completed isolated obj
updatedDep.Spec.Replicas = pointer.Int32(2)
patchData, err := patch.Data(updatedDep)


patchedDep, err := clientset.
                        AppsV1().Deployments("default").Patch(
                            ctx,
                            "dep1",
                            patch.Type(),
                            patchData,
                            metav1.PatchOptions{},
						)
```

Avail `PatchOptions`

- **DryRun**
- **Force**: field manager will acquire conflicting fields owned by other field managers.
- **FieldManager**
- **FieldValidation** (same as Creating a Resource)

Possible errors in `k8s.io/apimachinery/pkg/api/errors`

- `IsInvalid`
- `IsConflict`

## Server-side Apply

**++ fieldManager**

`Patch` method :cry: data must be passed in JSON format, error-prone.

```go
import "sigs.k8s.io/controller-runtime/pkg/client"

wantedDep := appsv1.Deployment{
    Spec: appsv1.DeploymentSpec{
    	Replicas: pointer.Int32(1),
    [...]
}
wantedDep.SetName("dep1")

// get K & V
wantedDep.APIVersion, wantedDep.Kind =
    appsv1.SchemeGroupVersion.
    	WithKind("Deployment").
    	ToAPIVersionAndKind()
    
patch := client.Apply
patchData, err := patch.Data(&wantedDep)
    
patchedDep, err := clientset.
	AppsV1().Deployments("default").Patch(
        ctx,
        "dep1",
        patch.Type(),
        patchData,
        metav1.PatchOptions{
        FieldManager: "my-program",
	},
)
```

`Apply` method, since v1.21, each manager owning a set of values in the resource spec, and only the fields the manager is responsible for.

```go
import (
	acappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
)

Apply(
    ctx context.Context,
    deployment *acappsv1.DeploymentApplyConfiguration,  // optional as pointers
    opts metav1.ApplyOptions,
) (result *v1.Deployment, err error)
```

Avail `ApplyOptions`

- **DryRun**
- **Force**: reacquire the conflicting fields owned by other managers.
- **FieldManager**

### Building ApplyConfiguration

**From Scratch**: kind/name/ns mandatory with helper function.

```go
func Deployment(name string, namespace string) *DeploymentApplyConfiguration {
    b := &DeploymentApplyConfiguration{}
    b.WithName(name)
    b.WithNamespace(namespace)
    b.WithKind("Deployment")
    b.WithAPIVersion("apps/v1")
    return b
}
```

```go
deploy1Config := acappsv1.Deployment(
    "deploy1",
    "default",
)
deploy1Config.WithSpec(acappsv1.DeploymentSpec())
deploy1Config.Spec.WithReplicas(2)

result, err := clientset.AppsV1().
    Deployments("default").Apply(
    	ctx,
    	deploy1Config,
    	metav1.ApplyOptions{
    		FieldManager: "my-manager",
    		Force: true,
    },
)
```

**From existing**: read the deployment from the cluster, then extract from it then set fields.

```go
gotDeploy1, err := clientset.AppsV1().
                        Deployments("default").Get(
                            ctx,
                            "deploy1",
                        	metav1.GetOptions{},
                        )
if err != nil {
	return err
}

deploy1Config, err := acappsv1.ExtractDeployment(
                        	gotDeploy1,
                        	"my-manager",
                        )
if err != nil {
	return err
}

If deploy1Config.Spec == nil {
	deploy1Config.WithSpec(acappsv1.DeploymentSpec())
}
deploy1Config.Spec.WithReplicas(2)

result, err := clientset.AppsV1().
	Deployments("default").Apply(
        ctx,
        deploy1Config,
        metav1.ApplyOptions{
        	FieldManager: "my-manager",
        	Force: true,
		},
)
```

## Watching Resources

```go
Watch(
    ctx context.Context,
    opts metav1.ListOptions,
) (watch.Interface, error)

type Interface interface {
    ResultChan() <-chan Event // recv only, for range to extract
    Stop()                    // stop watching
}

type Event struct {
    Type EventType            // Added/Modified/Deleted/Bookmark/Error
    Object runtime.Object
}
```

Avail `ListOptions`: see "Getting List of Resources"

```go
import "k8s.io/apimachinery/pkg/watch"

watcher, err := clientset.AppsV1().
    Deployments("project1").
    Watch(
        ctx,
        metav1.ListOptions{},
)
if err != nil {
	return err
}

for ev := range watcher.ResultChan() {
    // type assertion
    switch v := ev.Object.(type) {
	case *appsv1.Deployment:
        fmt.Printf("%s %s\n", ev.Type, v.GetName())
    case *metav1.Status:
		fmt.Printf("%s\n", v.Status)
		watcher.Stop()
    }
}   
```

## Errors & Statuses

Note: **Kind** could be either single name of the resource **or a list of**.

Functions are provided for this `StatusError` type to access the underlying `Status`.

- `Is<ReasonValue>(err error) bool`: whether err is of a particular status.
- `FromObject(obj runtime.Object) error`: build from when recv `metav1.Status` during watch.
- `(e *StatusError) Status() metav1.Status`: return underlying Status.
- `ReasonForError(err error) metav1.StatusReason`: return reason of underlying Status.
- `HasStatusCause(err error, name metav1.CauseType) bool`: whether had specific cause type.
- `StatusCause(err error, name metav1.CseType) (metav1.StausCause, bool)`: return cause of.
- `SuggestsClientDelay(err error) (int, bool)`: whether had RetryAfterSeconds.

```go
type StatusError struct {
	ErrStatus metav1.Status
}
```

```go
type Status struct {
    Status string          // Success or Failure
    Message string
    Reason StatusReason    // related to HTTP status code
    Details *StatusDetails
    Code int32             // HTTP status code
}

type StatusDetails struct {
    Name string
    Group string
    Kind string
    UID types.UID
    Causes []StatusCause    // enum the invalid fields & type of err for each field
    RetryAfterSeconds int32
}

type StatusCause struct {
    Type CauseType
    Message string
    Field string
}
```

## RESTClient

A REST client for each G/V

```go
// client of core/v1
restClient := clientset.CoreV1().RESTClient()
```

Interface impl

```go
type Interface interface {
    GetRateLimiter()          flowcontrol.RateLimiter
    Verb(verb string)         *Request
    Post()                    *Request
    Put()                     *Request
    Patch(pt types.PatchType) *Request
    Get()                     *Request
    Delete()                  *Request
    APIVersion()              schema.GroupVersion
}
```

### Building the Request

```bash
/apis/<group>/<version>          # group & version is fixed.
    /namespaces/<namesapce_name>
    	/<resource>
    		/<resource_name>
    			/<subresource>
```

Methods to build based on path

- `Namespace(namespace string) *Request`
- `NamespaceIfScoped(namespace string, scoped bool) *Request`
- `Resource(resource string) *Request`
- `Name(resourceName string) *Request`
- `SubResource(subresources ...string) *Request`
- `Prefix(segments ...string) *Request`
- `Suffix(segments...string) *Request`
- `AbsPath(segments ...string) *Request`

Methods complete the request with query param/body/headers.

- `Param(paramName, s string) *Request`
- `VersionedParams(obj runtime.Object, codec runtime.ParameterCodec,) *Request`
  - **obj** is generally `*Options`.
  - **codec** is generally `scheme.ParameterCodec`
- `SetHeader(key string, values ...string) *Request`
- `Body(obj interface{}) *Request`
  - **obj** could be string, []byte, io.Reader, runtime.Object (to be marshaled)

Meethods to configure the technical properties

- `BackOff(manager BackoffManager) *Request`: set Backoff manager for the request.
  - `rest.NoBackoff` (default)
  - `rest.URLBackoff`
  - any impl `rest.BackoffManager`
- `Throttle(limiter flowcontrol.RateLimiter) *Request`
- `MaxRetries(maxRetries int) *Request`
- `Timeout(d time.Duration) *Request`
- `WarningHandler(handler WarningHandler) *Request`

### Executing the Request

- `Do(ctx context.Context) Result`
- `Watch(ctx context.Context) (watch.Interface, error)`
- `Stream(ctx context.Context) (io.ReadCloser, error)`
- `DoRaw(ctx context.Context) ([]byte, error)`

### Exploiting the Reselt

`Result` obj returned after `Do()`

- `Into(obj runtime.Object) error`
- `Error() error`: 
- `Get() (runtime.Object, error)`
- `Raw() ([]byte, error)`
- `StatusCode(statusCode *int) Result`
- `WasCreated(wasCreated *bool) Result`
- `Warnings() []net.WarningHeader`

### Getting Result as a Table

`Accept` header when make a List operation.

```go
import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

restClient := clientset.CoreV1().RESTClient()  // get RESTClient

req := restClient.Get().
        Namespace("project1").   // indicate ns
        Resource("pods").        // indicate resources
        SetHeader(               // set required header
            "Accept",
            fmt.Sprintf(
                "application/json;as=Table;v=%s;g=%s",
                metav1.SchemeGroupVersion.Version,
                metav1.GroupName
            )
        )

var result metav1.Table         // to store the resule of the request
err = req.Do(ctx).Into(&result) // exec
if err != nil {
	return err
}

for _, colDef := range result.ColumnDefinitions { // iter col def
	// display header
}

for _, row := range result.Rows {       // iter row
    for _, cell := range row.Cells {    // iter cell of row
    	// display cell
    }
}
```

## Discovery Client

```bash
$ kubectl api-resources
```

```go
import "k8s.io/client-go/discovery"

// DiscoveryClient constructor
NewDiscoveryClientForConfig(c *rest.Config,) (*DiscoveryClient, error)
NewDiscoveryClientForConfigOrDie(c *rest.Config,) *DiscoveryClient
NewDiscoveryClientForConfigAndClient(c *rest.Config, httpClient *http.Client,) (*DiscoveryClient, error)
```

## RESTMapper

To map between REST Resources and Kubernetes Kinds.

`DefaultRESTMapper` as default impl, G/V/K must be added manully.

### PriorityRESTMapper

It gets all the groups served by the Kubernetes API & return the preferred version.

```go
import "k8s.io/client-go/restmapper"

discoveryClient := clientset.Discovery()
apiGroupResources, err := restmapper.GetAPIGroupResources(discoveryClient,)
if err != nil {
	return err
}
restMapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources,)
```

### DeferredDiscoveryRESTMapper

It uses a `PriorityRESTMapper` internally, but will **wait for** the first request to initialize the RESTMapper.

```go
import "k8s.io/client-go/restmapper"

discoveryClient := clientset.Discovery()
defRestMapper :=restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient),)
```

