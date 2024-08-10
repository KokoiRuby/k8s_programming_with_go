The Client-go Library provides a number of clients that can act with the Kubernetes API.

- `kubernetes.Clientset`: a set of clients, one for each G/V to exec op on resources.
- `rest.RESTClient`: a client to perform REST op on resources.
- `discovery.DiscoveryClient`: a client to discover resources.

## Fake Clientset

Tthe Client-go Library provides **fake** impl of these interfaces for utest. K8s API won't be called. No persisted in etcd, no validation & mutation will be performed, instead stored as-in an im-mem storage.

```go
func CreatePod(
	ctx context.Context,
	clientset kubernetes.Interface,  // cli impl
    name string,
	namespace string,
	image string,
) (pod *corev1.Pod, error)
```

```go
import "k8s.io/client-go/kubernetes/fake"

clientset := fake.NewSimpleClientset()
pod, err := CreatePod(
    context.Background(),
    clientset,
    aName,
    aNs,
    anImage,
)
```

```go
func TestCreatePod(t *testing.T) {
	var (
		name      = "a-name"
		namespace = "a-namespace"
		image     = "an-image"
		wantPod   = &corev1.Pod{
			ObjectMeta: v1.ObjectMeta{
				Name:      "a-name",
				Namespace: "a-namespace",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "runtime",
						Image: "an-image",
					},
				},
			},
		}
	)

	clientset := fake.NewSimpleClientset()
	gotPod, err := CreatePod(
		context.Background(),
		clientset,
		name,
		namespace,
		image,
	)

	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}

	if !reflect.DeepEqual(gotPod, wantPod) {
		t.Errorf("CreatePod() = %v, want %v", gotPod, wantPod)
	}
}
```

## Reacting to Actions

**Fake** clientset provides methods to add **Reactors**, func that are **executed when specific op are done** on resources. 

```go
type ReactionFunc func(
	action Action,
) (handled bool, ret runtime.Object, err error)
```

Except Watch & Proxy.

```go
type WatchReactionFunc func(
	action Action,
) (handled bool, ret watch.Interface, err error)

type ProxyReactionFunc func(
	action Action,
) (
	handled bool,
	ret restclient.ResponseWrapper,
	err error
)
```

The **Fake** field of the fake Clientset maintains several lists of **Reaction** functions in Chain

- `ReactionChain`
- `WatchReactionChain`
- `ProxyReactionChain`

Every time an op on a resource is invoked, the reactors are executed in chain-like fashion.

Chain will be terminated if returned false **handled**.

++ **Reaction**

- `*` for both = exec for op of any **verb** & any **resource**.
- `ReactionFunc` for additional filtering.

```go
AddReactor(verb, resource string, reaction ReactionFunc,)
PrependReactor(verb, resource string, reaction ReactionFunc,)

AddWatchReactor(resource string, reaction WatchReactionFunc,)
PrependWatchReactor(resource string,reaction WatchReactionFunc,)

AddProxyReactor(resource string, reaction ProxyReactionFunc,)
PrependProxyReactor(resource string, reaction ProxyReactionFunc,)
```

When the **fake** Clientset is created, a **Reactor** is added for both chains: `ReactionChain` and `WatchReactionChain`.

Example:

- Invoke a **Create** operation using the fake Clientset & stored in in-mem, a subsequent invocation of a
  **Get** operation on this resource will return the previously saved resource.
- If you do not want to use this default behavior, you can redefine the chains, make them empty.
- If you want to get some validation or some mutation on the passed resource, you can precede reactors using `PrependReactor` or `PrependWatchReactor`

```go
import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

clientset := fake.NewSimpleClientset()
// mutate to nodeName to node1
clientset.Fake.PrependReactor("create", 
                              "pods", 
                              func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
                                act := action.(ktesting.CreateAction)
                                ret = act.GetObject()
                                pod := ret.(*corev1.Pod)
                                pod.Spec.NodeName = "node1"
                                return false, pod, nil
})

pod, _ := CreatePod(
	context.Background(),
	clientset,
	name,
	namespace,
	image,
)
```

## Checking the Actions

The **mock** helps you **verify** that specific functions were called during the execution of the tested code.

The **fake** Clientset **registers** all the **Actions** done on it, you can **access** these **Actions** to check if match expected.

```go
actions := clientset.Actions()  // retrusn a list of obj that impl interface

type Action interface {
    GetNamespace() string
    GetVerb() string
    GetResource() schema.GroupVersionResource 
    Matches(verb, resource string) bool
    [...]
}
```

After you have checked the Verb of the Action, you can cast it to one of the interfaces related to the **Verb**: **GetAction, ListAction, CreateAction, UpdateAction, DeleteAction, DeleteCollectionAction, PatchAction, WatchAction, ProxyGetAction, and GenericAction**.

Interface provides more methods to get info specific to the Operation.

```go
import (
	"context"
	"reflect"
	"testing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestCreatePodActions(t *testing.T) {
	var (
		name       = "a-name"
		namespace  = "a-namespace"
		image      = "an-image"
		wantPod    = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "a-name",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "runtime",
						Image: "an-image",
					},
				},
			},
		}
		wantActions = 1
	)

    // 1. Create a fake Clientset
	clientset := fake.NewSimpleClientset()
	// 2. Call the CreatePod function to test
    _, _ = CreatePod(
		context.Background(),
		clientset,
		name,
		namespace,
		image,
	)

    // 3. Get the actions done during the execution of the function
	actions := clientset.Actions()
    // 4. Assert the number of actions
	if len(actions) != wantActions {
		t.Errorf("# actions = %d, want %d", len(actions), wantActions)
	}
	
    // 5. Get the first and only action done during execution
	action := actions[0]
    // 6. Assert the namespace value passed during the Action
	actionNamespace := action.GetNamespace()
	if actionNamespace != namespace {
		t.Errorf("action namespace = %s, want %s", actionNamespace, namespace)
	}

    // 7. Assert the Verb and Resource used for the Action
	if !action.Matches("create", "pods") {
		t.Errorf("action verb = %s, want create", action.GetVerb())
		t.Errorf("action resource = %s, want pods", action.GetResource().Resource)
	}

    // 8. Cast the Action to the CreateAction interface
	createAction := action.(ktesting.CreateAction)
    // 9. Assert the object value passed during the CreateAction
	obj := createAction.GetObject()
	if !reflect.DeepEqual(obj, wantPod) {
		t.Errorf("create action object = %v, want %v", obj, wantPod)
	}
}
```

## Fake REST Client

```go
import "k8s.io/client-go/rest/fake"

type RESTClient struct {
    NegotiatedSerializer runtime.NegotiatedSerializer  // codec K8s API
    GroupVersion schema.GroupVersion                   // specific to GV
    VersionedAPIPath string                            // prefix of API path
    Err error              // imitate the result of the request
    Req *http.Request      
    Client *http.Client    
    Resp *http.Response 
}
```

```go
import (
	"context"
	"errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest/fake"
)

// assert result of func returns an err
restClient := &fake.RESTClient{
	GroupVersion:         corev1.SchemeGroupVersion,
	NegotiatedSerializer: scheme.Codecs,
	Err:                  errors.New("an error from the rest client"),
}

pods, err := getPods(
	context.Background(),
	restClient,
	"default",
)

// assert result of func returns a NotFound 404
restClient := &fake.RESTClient{
    GroupVersion: corev1.SchemeGroupVersion,
    NegotiatedSerializer: scheme.Codecs,
    Err: nil,
    Resp: &http.Response{
    	StatusCode: http.StatusNotFound,
    },
}
```

## FakeDiscovery Client

```go
clientset := fake.NewSimpleClientset()
discoveryClient, ok :=
	clientset.Discovery().(*fakediscovery.FakeDiscovery)
if !ok {
	t.Fatalf("couldn't convert Discovery() to *FakeDiscovery")
}
```

**Subbing the ServerVersion**

To simulate the server version

```go
func checkMinimalServerVersion(
	clientset kubernetes.Interface,
	minMinor int,
) (bool, error) {
    discoveryClient := clientset.Discovery()
    info, err := discoveryClient.ServerVersion()
    if err != nil {
    	return false, err
    }
    major, err := strconv.Atoi(info.Major)
    if err != nil {
    	return false, err
    }
    minor, err := strconv.Atoi(info.Minor)
    if err != nil {
    	return false, err
    }
    return major == 1 && minor >= minMinor, nil
}
```

```go
func Test_getServerVersion(t *testing.T) {
    client := fake.NewSimpleClientset()
    fakeDiscovery, ok := client.Discovery().(*fakediscovery.FakeDiscovery)
    if !ok {
        t.Fatalf("couldn't convert Discovery() to *FakeDiscovery")
    }
    
    // assert 1.10
    fakeDiscovery.FakedServerVersion = &version.Info{
        Major: "1",
        Minor: "10",
    }
    
    res, err := checkMinimalServerVersion(client, 20)
    if res != true && err != nil {
        t.Error(err)
    }
}
```

**Acions**

To evaluate your functions’ use of these methods, you can **assert the actions** done by the Discovery client.

- ServerVersion – “get”, “version”
- ServerGroups – “get”, “group”
- ServerResourcesForGroupVersion – “get”, “resource”
- ServerGroupsAndResources – “get”, “group” and “get”, “resource”

**Mocking Resources**

To mock the result of these methods with various values of groups and resources, you can fill this **Resources** field before calling these methods.