Controller Manager embeds several Controllers and the role of each is to **watch** for instances of a specific **high-level** resource and **use low-level resources to implment** these high-level instances.

**Operator = Controller to manage K8s API Extensions.**

The **controller-runtime** Library leverages these tools to provide **abstractions** around the Controller pattern to help you write Operators.

```bash
$ go get sigs.k8s.io/controller-runtime@v0.13.0
```

## The Manager

It provides **shared resources** to all Controllers running within the manager

- A **client** for reading and writing resources
- A **cache** for reading resources from a local cache
- A **scheme** for registering all native and custom resources

`GetConfigOrDie()` provided by the **controller-runtime** lib instead of using Client-go. It will try to get conf

- `--kubeconfig` flag
- KUBECONFIG env
- in-cluser conf
- `$HOME/.kube/config`

If you want the controller to access CR, u will need to provide a **scheme** to resolve.

```go
import (
    "flag"
    "sigs.k8s.io/controller-runtime/pkg/client/config"
    "sigs.k8s.io/controller-runtime/pkg/manager"
    clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	mygroupv1alpha1 "github.com/myid/myresource-crd/pkg/apis/mygroup.example.com/v1alpha1"
    
)

scheme := runtime.NewScheme()
clientgoscheme.AddToScheme(scheme)  // ++ built-in
mygroupv1alpha1.AddToScheme(scheme) // ++ cr


// parse cmd line flag
flag.Parse() 
mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
    Scheme: scheme,
})
```

## The Controller

It is responsible for **implementing the Spec** given by the instances of a specific Kubernetes resource.

The Controller **watches** for specific resources, and receives **Watch events** (Create/Update/Delete), When events triggered, the Controller **populates a Queue** with a **Request** containing the name and namespace of the “primary resource” instance affected by the event.

Note: The Requests can be **batched** if multiple occur in a short time.

If the event is received by an instance of another resource, the **primary resource** is found by following **ownerReference**.

Example: When a ReplicaSet is modified, an Update event is received for this ReplicaSet, the Controller finds the
Deployment referenced by the updated ReplicaSet, using the **ownerReference** contained in the ReplicaSet. Then, the referenced Deployment instance is enqueued. 

The Controller implements a **Reconcile** method, which will be called every time a Request (name & ns of primary resource) is available in the queue.

```go
import (
	"sigs.k8s.io/controller-runtime/pkg/controller"
)
controller, err = controller.New("my-operator", mgr,controller.Options{
	Reconciler: reconcile.Func(reconcileFunction),
})

func reconcileFunction(ctx context.Context, r reconcile.Request) (reconcile.Result, error) {
    // TODO
    return reconcile.Result{}, nil
}
```

### Watching Resources

To indicate to the container which resources to watch

```go
Watch(
	src source.Source,
	eventhandler handler.EventHandler,
	predicates ...predicate.Predicate,
) error
```

Param#1 `source.Source`

- Impl#1 **Kind**: watch for events on Kubernetes objects of a specific kind.
- Impl#2 **Channel**: events outside the cluster

Param#2 `handler.EventHandler`

- Impl#1 **EnqueueRequestForObject** for primary resource handled by the controller.
- Impl#2 **EnqueueRequestForOwner** for resources owned by the primary resource.

Param#3 `predicate.Predicate`

- Impl#1 **Funcs** struct
- Impl#2 **func NewPredicateFuncs(filter func(object client.Object) bool,) Funcs**
  - accepts a filter function and returns a **Funcs** structure
- Impl#3 **ResourceVersionChangedPredicate** struct
  - only **Update** events will be filtered so that only the updates with a `metadata.resourceVersion` change will be processed.
- Impl#4 **GenerationChangedPredicate** struct
  - only **Update** events will be filtered so that only the updates with a `metadata.Generation` change will be processed.
- Impl#5 AnnotationChangedPredicate struct
  - only **Update** events will be filtered so that only the updates with a `metadata.Annotations` change will be processed.

```go
// watch for Deployment
controller.Watch(
    &source.Kind{
    	Type: &appsv1.Deployment{},
	},
)

// watch & handle CR
controller.Watch(
    &source.Kind{
    	Type: &mygroupv1alpha1.MyResource{},
    },
    &handler.EnqueueRequestForObject{},
)

// if the Controller handles the MyResource primary resource and is creating Pods to implement MyResource
// watch pod where CR is owner
controller.Watch(
    &source.Kind{
    	Type: &corev1.Pod{},
    },
	&handler.EnqueueRequestForOwner{
    	OwnerType: &mygroupv1alpha1.MyResource{},
		IsController: true,
	},
)
```

```go
type Funcs struct {
    // Create returns true if the Create event should be processed
    CreateFunc func(event.CreateEvent) bool
    // Delete returns true if the Delete event should be processed
    DeleteFunc func(event.DeleteEvent) bool
    // Update returns true if the Update event should be processed
    UpdateFunc func(event.UpdateEvent) bool
    // Generic returns true if the Generic event should be processed
    GenericFunc func(event.GenericEvent) bool
}
```

### Using the Controller Builder

**Easier** to create a Controller.

```go
import (
	"sigs.k8s.io/controller-runtime/pkg/builder"
)

// to initiate a new ControllerBuilder
func ControllerManagedBy(m manager.Manager) *Builder
```

`For(object client.Object, opts ...ForOption) *Builder` to **indicate the primary resource** handled by the controller.

`Owns(object client.Object, opts ...OwnsOption) *Builder` to **indicate a resource owned** by the controller.

`Watches(src source.Source, eventhandler handler.EventHandler, opts ...WatchesOption) *Builder` to add more watchers not covered by the For or Owns.

`WithEventFilter(p predicate.Predicate) *Builder` to add predicates common to all watchers created with For, Owns, and Watch methods.

`WithOptions(options controller.Options) *Builder` sets the options that will be passed internally to the controller.New function.

`WithLogConstructor(func(*reconcile.Request) logr.Logger) *Builder` sets the logConstructor option.

`Named(name string) *Builder` sets the name of the constructor.

`Build(r reconcile.Reconciler,) (controller.Controller, error)` builds and returns the Controller.

`Complete(r reconcile.Reconciler) error` build only, no return.

### Injecting Manager Resources into the Reconciler

The Reconcile function needs access to these shared resources = client/cache/scheme provided by The Manager.

1. **Passing the values**: passing an instance of a Reconcile struct impl Reconciler interface.

```go
mgr, err := manager.New(
		config.GetConfigOrDie(),
		manager.Options{
			manager.Options{
				Scheme: scheme,
			},
		},
)

// get client/cache/scheme from manager
mgrClient := mgr.GetClient()
mgrCache := mgr.GetCache()
mgrScheme := mgr.GetScheme()

type MyReconciler struct {
    client client.Client
	cache cache.Cache
	scheme *runtime.Scheme
}

err = builder.
		ControllerManagedBy(mgr).
		For(&mygroupv1alpha1.MyResource{}).
		Owns(&corev1.Pod{}).
		Complete(&MyReconciler{
            client: mgr.GetClient(),
			cache: mgr.GetCache(),
			scheme: mgr.GetScheme(),
        })
```

2. **Injectors**: The Reconciler impl need to implement the specific injector interfaces from the inject package: `inject.Client`, `inject.Cache`, `inject.Scheme`, and so on. They will be called at init time, when you call `controller.New` or `builder.Complete`.

```go
type MyReconciler struct {
    client client.Client
    cache cache.Cache
    scheme *runtime.Scheme
}
func (a *MyReconciler) InjectClient(c client.Client,) error {
    a.client = c
    return nil
}
func (a *MyReconciler) InjectCache(c cache.Cache,) error {
    a.cache = c
    return nil
}
func (a *MyReconciler) InjectScheme(s *runtime.Scheme,) error {
    a.scheme = s
    return nil
}
```

## Using the Client

To R/W resources on the cluster.

The **Read** methods internally use a **Cache system**, based on **Informers** and **Listers** to limit read accesss to API Server.

Note: objects returned by Read operations are **pointers** to values into the Cache. You must **never modify** these objects directly. Instead, you must create a **deep copy** of the returned objects before modifying them.

### Getting Information

```go
Get(
    ctx context.Context,
    key ObjectKey,         // indicate the namespace and name of the resource
    obj Object,            // indicate the Kind f the resource 
    opts ...GetOption,
) error

myresource := mygroupv1alpha1.MyResource{}
err := a.client.Get(ctx, req.NamespacedName, &myresource)
// pass a specific resourceVersion value to the Get
err := a.client.Get(ctx, 
                    req.NamespacedName, 
                    &myresource, 
                    &client.GetOptions{
    					Raw: &metav1.GetOptions{
        					ResourceVersion: "0",
    					},
					},
)
```

### Listing

ListOption - `client.*`

- **InNamespace** alias to string, return the resources of a specific namespace.
- **MatchingLabels**, alias to map[string]string.
- **HasLabels**, alias to []string.
- **MatchingLabelsSelector** embedding a labels.Selector interface.
- **MatchingFields**, alias to fields.Set.
- **MatchingFieldsSelector** embedding a fields.Selector.
- **Limit** alias to int64 and Continue, alias to string, to paginate the result.

```go
List(
    ctx context.Context,
    list ObjectList,     // indicating the Kind of the resource to list
    opts ...ListOption,
) error

// **MatchingLabels**, alias to map[string]string
matchLabel := client.MatchingLabels{
	"app": "myapp",
}

// **HasLabels**, alias to []string
hasLabels := client.HasLabels{"app", “debug”}

// **MatchingLabelsSelector** embedding a labels.Selector interface
selector := labels.NewSelector()
require, err := labels.NewRequirement(
	"mykey",
	selection.NotEquals,
	[]string{"ignore"},
)
selector = selector.Add(*require)
labSelOption := client.MatchingLabelsSelector{
	Selector: selector,
}

// **MatchingFields**, alias to fields.Set
matchFields := client.MatchingFields{
	"status.phase": "Running",
}

// **MatchingFieldsSelector** embedding a fields.Selector
fieldSel := fields.OneTermNotEqualSelector(
    "status.phase",
    "Running",
)
fieldSelector := client.MatchingFieldsSelector{
	Selector: fieldSel,
}
```

### Creating

CreateOption

- **DryRunAll** indicates all the operations should be executed except those persisting the resource to storage.
- **FieldOwner** alias to string, indicates the name of the field manager.

```go
Create(
    ctx context.Context,
    obj Object,          // defines the kind of object to create
    opts ...CreateOption,
) error

podToCreate := corev1.Pod{ [...] }
podToCreate.SetName("nginx")
podToCreate.SetNamespace("default")
err = a.client.Create(ctx, &podToCreate)
```

### Deleting

```go
Delete(
    ctx context.Context,
    obj Object, k         // defines the kind of object to delete
    opts ...DeleteOption,
) error
```

DeleteOption

- **DryRunAll** indicates all the operations should be executed except those persisting the resource to storage.
- **GracePeriodSeconds** alias to int64.
- **Preconditions** alias to metav1.Preconditions, indicates which resource u expect to delete, such as UID, ResourceVersion.
- **PropagationPolicy** alias to metav1.DeletionPropagation, indicates how GC will be performed.

### Deleting a collection of

```go
DeleteAllOf(
    ctx context.Context,
    obj Object,                // defines the kind of object to delete
    opts ...DeleteAllOfOption, // a combo of ListOption & DeleteOption
) error
```

### Updating

UpdateOption

- **DryRunAll** indicates all the operations should be executed except those persisting the resource to storage.
- **FieldOwner** alias to string, indicates the name of the field manager.

```go
Update(
    ctx context.Context,
    obj Object,
    opts ...UpdateOption,
) error
```

### Patching

PatchOption

- **DryRunAll** indicates all the operations should be executed except those persisting the resource to storage.
- **FieldOwner** alias to string, indicates the name of the field manager.
- **ForceOwnership** alias to struct{}, indicates that the caller will reacquire the conflicting fields owned by other managers.

```go
Patch(
	ctx context.Context,
    obj Object,
    patch Patch,
    opts ...PatchOption,
) error
```

1. **Server-side Apply**

- Need to specify a patch value of client.Apply.
- The obj value must be the new object definition including name/namespace/gvk.
- The name of the field manager is required and must be specified with the option `client.FieldOwner(name)`.

```go
deployToApply := appsv1.Deployment{ [...] }
deployToApply.SetName("nginx")
deployToApply.SetNamespace("default")
deployToApply.SetGroupVersionKind(
	appsv1.SchemeGroupVersion.WithKind("Deployment"),
)
err = a.client.Patch(
    ctx,
    &deployToApply,
    client.Apply,
    client.FieldOwner("mycontroller"),
    client.ForceOwnership,
)
```

2. Strategic Merge Patch

- Still must read the resource from the cluster, create a Patch using the **StrategicMergeFrom** function.

```go
var deploymentRead appsv1.Deployment

// get deploy to patch
err = a.client.Get(
    ctx,
    key,                     // an ObjectKey defining the namespace and name
    &deploymentRead)
if err != nil {
	return reconcile.Result{}, err
}

// create deepcopy
patch := client.StrategicMergeFrom(deploymentRead.DeepCopy())

// deepcopy & modify
depModified := deploymentRead.DeepCopy()
depModified.Spec.Template.Spec.Containers[0].Env = append(depModified.Spec.Template.Spec.Containers[0].Env, 
       corev1.EnvVar{
		Name: "newName",
		Value: "newValue",
		}
      )

// patch
err = a.client.Patch(ctx, &depModified, patch)
```

3. Merge Patch

- A similar way to a Strategic Merge Patch, except for how the lists are merged.
- The original lists are not considered, and the new list is the list defined in the patch.

```go
patch := client.MergeFrom(deploymentRead.DeepCopy())
```

### Updating the Status

When u want to modify values into the Status part of the CR to indicate the current status of it.

CR **must** declare the status field in the list of sub-resources.

```go
Update(
    ctx context.Context,
    obj Object,
    opts ...UpdateOption,
) error

err = client.Status().Update(ctx, obj)
```

### Patching the Status

Same as the Patch, except that it will patch the Status part of the resource only.

```go
Patch(
    ctx context.Context,
    obj Object,
    patch Patch,
    opts ...PatchOption,
) error

err = client.Status().Patch(ctx, obj, patch)
```

## Logging

Initialized with a call to SetLogger.

The **Logger** is a “structured” logging system, in that logs are mostly made of k/v pairs:

- `level` verbosity level
- `ts` timestamp of the log entry
- `logger` name of the logger
- `msg ` message associated with the log entry
- `error` the error associated with the log entry

```go
import (
    crlog "sigs.k8s.io/controller-runtime/pkg/log"
    "sigs.k8s.io/controller-runtime/pkg/log/zap"
)
func main() {
    log.SetLogger(zap.New())
    log.Log.Info("starting")
    ...
}
```

### Verbosity

`log.V(n).Info(...)`

### Predefined Values

Build with predefined k/v by `WithValues`.

```go
ctrlLog := log.Log.WithValues("package”, “controller”)
```

### Name

```go
ctrlMainLog = ctrlLog.Log.WithName("main")
```

### Get Logger from Context

Extract the logger from the Context using the function `FromContext`.

```go
log := log.FromContext(ctx).WithName("reconcile")
```

## Events

`kubectl describe ...`

Events are **sent by controllers** to inform the user that some event occurred related to an object.

To send such events **from the Reconcile function**, you need to have access to the **EventRecorder** instance provided by the Manager.

```go
// embed EventRecorder in reconciler
type MyReconciler struct {
      client        client.Client
      EventRecorder record.EventRecorder
}

func main() {
      ...
      // get from manager
      eventRecorder := mgr.GetEventRecorderFor("MyResource")
      err = builder.
            ControllerManagedBy(mgr).
            Named(controller.Name).
            For(&mygroupv1alpha1.MyResource{}).
            Owns(&appsv1.Deployment{}).
            // set into reconciler
    		Complete(&controller.MyReconciler{
                  EventRecorder: eventRecorder,
            })
      [...]
}

// call inside reconcile func
func (record.EventRecorder) Event(
     // indicates to which object to attach the Event
     object runtime.Object, corev1.EventTypeWarning
     // corev1.EventTypeNormal & 
     eventtype string, 
     // short value in UpperCamelCase format
     reason string,
     // to pass a static message
     message string,
)

// create a message using Sprintf
func (record.EventRecorder) Eventf(
     object runtime.Object,
     eventtype, reason, messageFmt string,
     args ...interface{},
)

// attach annotations to the event.
func (record.EventRecorder) AnnotatedEventf(
     object runtime.Object,
     annotations map[string]string,
     eventtype, reason, messageFmt string,
     args ...interface{},
)
```

