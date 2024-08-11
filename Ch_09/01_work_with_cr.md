## Generating a Clientset

[code-generator](https://github.com/kubernetes/code-generator): Golang code-generators used to implement [Kubernetes-style API types](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md).

First write the Go **structs** for the **kinds** defined by the CR.

```bash
$ mkdir -p github.com/myid/myresource-crd
$ cd github.com/myid/myresource-crd
$ go mod init github.com/myid/myresource-crd
$ mkdir -p pkg/apis/mygroup.example.com/v1alpha1/
$ cd pkg/apis/mygroup.example.com/v1alpha1/
```

`types.go` that contains the definitions of the **strcuts** for the **kinds**.

```go
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MyResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MyResourceSpec `json:"spec"`
}
type MyResourceSpec struct {
	Image  string            `json:"image"`
	Memory resource.Quantity `json:"memory"`
}
type MyResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MyResource `json:"items"`
}

```

### **Using deepcopy-gen** (TODO)

To generate a `DeepCopyObject()` method for each Kind structure, which is needed for these types to implement
the `runtime.Object` interface.

```bash
$ go install k8s.io/code-generator/cmd/deepcopy-gen@v0.24.4
$ sudo ln -s $GOPATH/bin/deepcopy-gen /usr/local/bin
```

It needs **annotations** to work. 

`//+k8s:deepcopy-gen=package` asks deepcopy-gen to generate deepcopy methods for all structures of the package.

`doc.go`

```go
// pkg/apis/mygroup.example.com/v1alpha1/doc.go     
// +k8s:deepcopy-gen=package                     
package v1alpha1
```

By default, deepcopy-gen will generate the `DeepCopy()` and `DeepCopyInto()` methods, but no `DeepCopyObject()`.

`//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object` for each struct.

```go
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MyResource struct {}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MyResourceList struct {}
```

Run

```bash
$ deepcopy-gen --input-dirs github.com/myid/myresource-crd/pkg/apis/mygroup.example.com/v1alpha1 \
-O zz_generated.deepcopy \
--output-base ../../.. \
--go-header-file ./hack/boilerplate.go.txt
```

### Using client-gen

To generate the clientset for the group/version.

```bash
$ go install k8s.io/code-generator/cmd/client-gen@v0.24.4
$ sudo ln -s $GOPATH/bin/client-gen /usr/local/bin
```

It needs **annotations** to work.

- `// +genclient`: ask client-gen to generate a Clientset for a **namespaced** resource.
- `// +genclient:nonNamespaced`: generate a Clientset for a **non-namespaced** resource.
- `// +genclient:onlyVerbs=create,get`: generate these verbs only, instead of generating all verbs by default.
- `// +genclient:skipVerbs=watch`: generate all verbs except these ones, instead of all verbs by default.
- `// +genclient:noStatus`: if a **Status** field is present in the annotated structure, an updateStatus function will be generated.

```go
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
type MyResource struct {}
```

`register.go` defines CR G/V.

```go
package v1alpha1

import (
     metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
     "k8s.io/apimachinery/pkg/runtime"
     "k8s.io/apimachinery/pkg/runtime/schema"
)

const GroupName = "mygroup.example.com"            // group name

var SchemeGroupVersion = schema.GroupVersion{
     Group: GroupName,
     Version: "v1alpha1",                         // version name
} 
var (
     SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
     localSchemeBuilder = &SchemeBuilder
     AddToScheme        = localSchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
     scheme.AddKnownTypes(SchemeGroupVersion,    // the list of resources to reg to the scheme
          &MyResource{},                         
          &MyResourceList{},
     )
     metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
     return nil
}
```

Run

Note: It is **recommended** to place this command in a **Makefile** to automatically run it every time the files defining the custom resource are modified.

```bash
$ client-gen \
     --clientset-name clientset \
     --input-base "" \
     --input github.com/myid/myresource-crd/pkg/apis/mygroup.example.com/v1alpha1 \
     --output-package github.com/myid/myresource-crd/pkg/clientset \
     --output-base ../../.. \
     --go-header-file hack/boilerplate.go.txt
```

Tree

```bash
$ tree
.
├── go.mod
├── go.sum
├── hack
│   └── boilerplate.go.txt
└── pkg
    ├── apis
    │   └── mygroup.example.com
    │       └── v1alpha1
    │           ├── doc.go
    │           ├── register.go
    │           ├── types.go
    │           └── zz_generated.deepcopy.go
    └── clientset
        └── clientset
            ├── clientset.go
            ├── doc.go
            ├── fake
            │   ├── clientset_generated.go
            │   ├── doc.go
            │   └── register.go
            ├── scheme
            │   ├── doc.go
            │   └── register.go
            └── typed
                └── mygroup.example.com
                    └── v1alpha1
                        ├── doc.go
                        ├── fake
                        │   ├── doc.go
                        │   ├── fake_mygroup.example.com_client.go
                        │   └── fake_myresource.go
                        ├── generated_expansion.go
                        ├── mygroup.example.com_client.go
                        └── myresource.go
```

### Using the Generated (fake) Clientset

Clientset is generated and the types implement the `runtime.Object` interface. U could manage CR like native K8s res.

```go
import (
     "context"
     "fmt"
     "github.com/myid/myresource-crd/pkg/clientset/clientset"
     metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
     "k8s.io/client-go/tools/clientcmd"
)

config, err :=
     clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
          clientcmd.NewDefaultClientConfigLoadingRules(),
          nil,
     ).ClientConfig()
if err != nil {
     return err
}

clientset, err := clientset.NewForConfig(config)
if err != nil {
     return err
}

list, err := clientset.MygroupV1alpha1().
     MyResources("default").
     List(context.Background(), metav1.ListOptions{})
if err != nil {
     return err
}

for _, res := range list.Items {
     fmt.Printf("%s\n", res.GetName())
}

```

## Using the Unstructured Package and Dynamic Client

`unstructured` package of the API Machinery

```go
import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)
```

It is possible to define any K8s resource **without** having to use the typed structures.

```go
type Unstructured struct {
    // Object is a JSON compatible map with
    // string, float, int, bool, []interface{}, or
    // map[string]interface{}
    // children.
    Object map[string]interface{}
}
```

`Getters` and `Setters` methods are defined for this type to access generic fields from the `TypeMeta` and `ObjectMeta` fields.

```go
// TypeMeta
GetAPIVersion() string
GetKind() string
GroupVersionKind() schema.GroupVersionKind

SetAPIVersion(version string)
SetKind(kind string)
SetGroupVersionKind(gvk schema.GroupVersionKind)

// ObjectMeta
GetName() string
SetName(name string).
// ...
```

Methods for Creating and Converting

```go
NewEmptyInstance() runtime.Unstructured                 // constructor with only apiVersion & kind copied of recv
MarshalJSON() ([]byte, error)                           // JSON repr of recv
UnmarshalJSON(b []byte) error                           // populate recv with passed JSON
UnstructuredContent() map[string]interface{}            // returns the val of Object field of recv
SetUnstructuredContent(content map[string]interface{},) // set the Object filed of recv
IsList() bool                                           // returns true if recv desc a list
ToList() (*UnstructuredList, error)                     // convert recv to an UnstructuredList
```

**Helper** functions to get/set the value of specific fields.

```go
// removes the requested field
RemoveNestedField
// returns a copy or the original value of the requested field
NestedFieldCopy, NestedFieldNoCopy
// gets and sets bool / float64 / int64 / string field
NestedBool, NestedFloat64, NestedInt64, NestedString, SetNestedField
// gets and sets fields of type map[string]interface{}
NestedMap, SetNestedMap
// gets and sets fields of type []interface{}
NestedSlice, SetNestedSlice
// gets and sets fields of type map[string]string
NestedStringMap, SetNestedStringMap
// ets and sets fields of type []string
NestedStringSlice, SetNestedStringSlice
```

Example

```go
import (
     myresourcev1alpha1 "github.com/myid/myresource-crd/pkg/apis/mygroup.example.com/v1alpha1"
     "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getResource() (*unstructured.Unstructured, error) {
     myres := unstructured.Unstructured{}
     myres.SetGroupVersionKind(
          myresourcev1alpha1.SchemeGroupVersion.
               WithKind("MyResource"))
     myres.SetName("myres1")
     myres.SetNamespace("default")
     err := unstructured.SetNestedField(
          myres.Object,
          "nginx",
          "spec", "image",
     )
     if err != nil {
          return err
     }
     // Use int64
     err = unstructured.SetNestedField(
          myres.Object,
          int64(1024*1024*1024),
          "spec", "memory",
     )
     if err != nil {
          return err
     }
     // or use string
     err = unstructured.SetNestedField(
          myres.Object,
          "1024Mo",
          "spec", "memory",
     )
     if err != nil {
          return err
     }
     return &myres, nil
}
```

`UnstructuredList` is defined as a structure containing an Object field and a slice of `Unstructured`.

```go
type UnstructuredList struct {
	Object map[string]interface{}
    // Items is a list of unstructured objects.
	Items []Unstructured
}
```

Getter/Setter to TypeMeta & ListMeta

```go
// TypeMeta 
GetAPIVersion() string
GetKind() string
GroupVersionKind() schema.GroupVersionKind

SetAPIVersion(version string)
SetKind(kind string)
SetGroupVersionKind(gvk schema.GroupVersionKind)

// ListMeta
GetResourceVersion() string
GetContinue() string
GetRemainingItemCount() *int64

SetResourceVersion(version string)
SetContinue(c string)
SetRemainingItemCount(c *int64)
```

Methods for Creating & Converting

```go
// creates a new instance of an Unstructured object using apiVersion and kind copied from the List recv.
NewEmptyInstance() runtime.Unstructured
// returns the JSON representation of the recv.
MarshalJSON() ([]byte, error)
// populates the recv with the passed JSON representation.
UnmarshalJSON(b []byte) error
// gets the value of the Object field of the recv
UnstructuredContent() map[string]interface{}
// executes the fn function for each item of the list
EachListItem(fn func(runtime.Object) error) error
```

### Convert btw Typed & Unstructured Objects

````go
import (
	"k8s.io/apimachinery/pkg/runtime"
)

converter := runtime.DefaultUnstructuredConverter // 1. get converter

var pod corev1.Pod
converter.FromUnstructured(           // convert unstructured obj to typed pod
	u.UnstructuredContent(), &pod,
)

var u unstructured.Unstructured       // convert typed pod to unstructured obj
u.Object = converter.ToUnstructured(&pod)
````

## The (fake) Dynamic Client

Dynamic Client, to work with untyped resources, described with the `Unstructured` type.

```go
// eturns a dynamic client, using the provided rest.Config
func NewForConfig(c *rest.Config) (Interface, error)
// panics in case of error
func NewForConfigOrDie(c *rest.Config) Interface
// returns a dynamic client, using the provided rest. Config, and the provided httpClient
NewForConfigAndClient(c *rest.Config, httpClient *http.Client,) (Interface, error)
```

Dynamic Client implements `dynamic.Interface`. It returns **Resource(gvr)** which implements `NamespaceableResourceInterface`.

```go
type Interface interface {
	Resource(resource schema.GroupVersionResource)
	NamespaceableResourceInterface
}

type NamespaceableResourceInterface interface {
    Namespace(string) ResourceInterface
    ResourceInterface
}

type ResourceInterface interface {
    Create(...)
    Update(...)
    UpdateStatus(...)
    Delete(...)
    DeleteCollection(...)
    Get(...)
    List(...)
    Watch(...)
    Patch(...)
    Apply(...) 
    ApplyStatus(...)
}
```

Example

```go
import (
    "context"
    "github.com/feloy/myresource-crd/pkg/apis/mygroup.example.com/v1alpha1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/dynamic"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func CreateMyResource(
    dynamicClient dynamic.Interface,
    u *unstructured.Unstructured,
) (*unstructured.Unstructured, error) {
    gvr := v1alpha1.SchemeGroupVersion.WithResource("myresources")
    return dynamicClient.Resource(gvr).Namespace("default").Create(context.Background(), u, metav1.CreateOptions{})
}
```

Fake Dynamic Client

```go
func NewSimpleDynamicClient(scheme *runtime.Scheme, objects ...runtime.Object,) *FakeDynamicClient)
```

```go
func TestCreateMyResourceWhenResourceExists(t *testing.T) {
    myres, err := getResource()
    if err != nil {
        t.Error(err)
    }

    dynamicClient := fake.NewSimpleDynamicClient(runtime.NewScheme(), myres)

    // Not really used, just to show how to use it
    dynamicClient.Fake.PrependReactor("create", "myresources", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
        return false, nil, nil
    })

    _, err = CreateMyResource(dynamicClient, myres)
    if err == nil {
        t.Error("Error should happen")
    }

    actions := dynamicClient.Fake.Actions()
    if len(actions) != 1 {
        t.Errorf("# of actions should be %d but is %d", 1, len(actions))
    }
}
```

