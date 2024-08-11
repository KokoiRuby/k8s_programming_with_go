## API Library Sources and Import

APIs are organized into **GVR**. Objects exchanged btw client & API Server are defined as **Kinds**. Hence **GVK**.

API [src](https://github.com/kubernetes/api).

```go
import "k8s.io/api/<group>/<version>"
```

## Content of a Package

Example: [k8s.io/api/apps/v1](https://github.com/kubernetes/api/tree/master/apps/v1)

- `types.go`: it defines all the **Kind** struct and other related sub-struct, also const & enum.
  - `Deployment`
    - `DeploymentSpec`
    - `DeploymentStatus`
      - `[]DeploymentCondition`
        - `DeploymentConditionType`

- `register.go`: it defines the **group and version** related to this package.
  - `SchemeGroupVersion` is group version used to **register** these objects.
  - `(SchemeBuilder.)AddToScheme` method: **add** the group, version,and Kinds **to a Scheme**.
- `doc.go`: contains the instructions to **generate files by generator**.
  - `generated.pb.go`
  - `types_swagger_doc_generated.go`
  - `zz_generated.deepcopy.go`

## Specific Content in [core/v1](https://github.com/kubernetes/api/tree/master/core/v1)

In addition to Kind structures, **utility methods** for specific types can be useful.

- `ObjectReference`: **refer** to any object in a unique way. Methods:
  - `SetGroupVersionKind`: set the fields APIVersion and Kind based on GVK.
  - `GroupVersionKind`: return a GVK.
  - `GetObjectKind`: return a ObjectKind.
- `ResourceList`: a map (resource name, quantity) → limits & requests.
  - `Cpu()`
  - `Memory()`
  - `Storage()`
  - `Pods()`
  - `StorageEphemeral()`

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: mypod
spec:
  containers:
    - name: runtime
      resources:
        requests:
          cpu: "250m"
          memory: "64Mi"
        limits:
          cpu: "500m"
          memory: "128Mi"
```

```go
requests := corev1.ResourceList{
	corev1.ResourceCPU:    *resource.NewQuantity(250, resource.BinarySI),
	corev1.ResourceMemory: *resource.NewQuantity(64*1024*1024, resource.BinarySI),
}

limits := corev1.ResourceList{
	corev1.ResourceCPU:    *resource.NewQuantity(500, resource.BinarySI),
	corev1.ResourceMemory: *resource.NewQuantity(128*1024*1024, resource.BinarySI),
}
```

- `Taint`: applied to Nodes to ensure that pods to be scheduled on.
  - TaintEffect (enum)
    - NoSchedule
    - PreferNoSchedule
    - NoExecute
  - `ToString()`: conver to `<key>=<value>:<effect>` as label.
- `Toleration`: applied to Pods to make it tolerate taints in Nodes.
  - TolerationOperator
    - Exists
    - Equal
  - `MatchToleration`: returns true if the two tolerations have the same values for Key, Operator, Value, and Effect.
  - `ToleratesTaint`: returns true if the toleration tolerates the Taint.
- `Well-Known Labels`: well-known keys in const that are used and their usage, such as `kubernetes.io/hostname`.

## Writing K8s Resources in Go

client-go is preferrable.

To create or update a resource, you will need to create the structure for the **Kind associated** with the resource.

Initiate a Deployment struct to create a Deployment **kind**.

```go
import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
)

func main() {
	// init a Deployment struct
	myDep := appsv1.Deployment{}
	fmt.Printf("%+v\n", myDep)
}
```

All structures related to **Kinds** first embed two generic structs in `/pkg/apis/meta/v1` of API machinery [lib](https://github.com/kubernetes/apimachinery):

- TypeMeta
- ObjectMeta

```go
// APIMachinery infers these values from the type of the struct by maintaining a Scheme: a map of GVK → Go struct
type TypeMeta struct {
    Kind string
    APIVersion string
}

// APIMachinery defines Getters and Setters for fields of this structure.
// ObjectMeta is embedded in Kind, so we could use these methods.
Type ObjectMeta {
    Name string
    GenerateName string
    Namespace string
    UID types.UID
    ResourceVersion string
    Generation int64
    Labels map[string]string
    Annotations map[string]string
    OwnerReferences []OwnerReference
    [...]
}
```

**Name**

```bash
myCM := corev1.ConfigMap{}
myCM.SetName("myConfigMap")
```

**Namespace**: does not need setter as we will specify in URL path.

```bash
$ curl $HOST/api/v1/namespaces/{over_here}/pods
```

**UID**: a unique **identifier** of resource in cluster, set by control plane & never get updated.

**ResourceVersion**: changed when update. Optimistic concurrency control: API Server will reject if not matched.

**Generation**: sequence number of resource controller to indicate the version of desired state, changed when spec update.

**Labels**

- `GetLabels() map[string]string`
- `SetLabels(labels map[string]string)`
- `GetAnnotations() map[string]string`
- `SetAnnotations(annotations map[string]string)`

```go
// construct label by go built-in map
myLabel1 := map[string]string{
	"app.kubernetes.io/component": "my-component",
	"app.kubernetes.io/name":      "my-app",
}

// construct label by apimachinery
myLabel2 := labels.Set{
	"app.kubernetes.io/component": "my-component",
	"app.kubernetes.io/name":      "my-app",
}
```

**OwnerReferences**: indicate resource is owned by another. Deployment → ReplicaSet.

**It is used widely when developing controllers and operators.** A controller or operator creates some resources to implement the specifications described by another resource, and it places an **OwnerReference** into the created resources, pointing to the resource giving the specifications.

```go
// need to construct by setting these fields
type OwnerReference struct {
	APIVersion string        
	Kind string
	Name string
	UID types.UID            // To know the UID of the obj to ref, need to get obj from k8s API using get/list
    Controller *bool         // A controller/operator must set this val to true 
	BlockOwnerDeletion *bool 
}
```

- Setting APIVersion and Kind

```go
// setup a clientset
clientSet, err := getClientSet()
if err != nil {
    panic(err)
}

// get pod
pod, err := clientSet.CoreV1().Pods("myns").Get(context.TODO(), "mypodname", metav1.GetOptions{})
if err != nil {
    panic(err)
}

// Solution 1: set the APIVersion and Kind of the Pod then copy all info from the pod
pod.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Pod"))
ownerRef := metav1.OwnerReference{
    APIVersion: pod.APIVersion,
    Kind:       pod.Kind,
    Name:       pod.GetName(),
    UID:        pod.GetUID(),
}
_ = ownerRef

// Solution 2: Copy name and UID from pod then set APIVersion and Kind on the OwnerReference
ownerRef = metav1.OwnerReference{
    Name: pod.GetName(),
    UID:  pod.GetUID(),
}
ownerRef.APIVersion, ownerRef.Kind = corev1.SchemeGroupVersion.WithKind("Pod").ToAPIVersionAndKind()
```

- Setting Controller, must be a bool ptr.

```go
// Solution 1: declare a value and use its address
controller := true
ownerRef.Controller = &controller

// Solution 2: use the BoolPtr function
ownerRef.Controller = pointer.BoolPtr(controller)
```

- Setting BlockOwnerDeletion
  - Propagation Policy on Delete
    - Orphan: not deleted by GC
    - Background: non-blocking GC
    - Foreground: blocking GC
  - if you are writing a controller or another process that **needs to wait for all the**
    **owned resources to be deleted**, the process will need to set the BlockOwnerDeletion to true on all the owned resources and to use the **Foreground propagation policy** when deleting the owner resource.

### Spec & Status

**Spec**: indicates the desired state by the user. 

**Reconcile Loop**: **Spec** read by *Controller to verb on resource, *Controller will retrive the status & set to **Status**.

### YAML Manifest

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
  labels:
    - component: my-component
spec:
  containers:
    - image: nginx
      name: nginx
```

```go
pod := corev1.Pod{
    ObjectMeta: metav1.ObjectMeta{
        Name: "nginx",
        Labels: map[string]string{
            "component": "mycomponent",
        },
    },
    Spec: corev1.PodSpec{
        Containers: []corev1.Container{
            {
                Name: "runtime",
                Image: "nginx",
            },
        },
    },
}

// or
pod1 := corev1.Pod{
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "runtime",
				Image: "nginx",
			},
		},
	},
}
pod1.SetName("my-pod")
pod1.SetLabels(map[string]string{
	"component": "my-component",
})
```