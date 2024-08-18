**CR** permits extension of the K8s API.

**Clientset & DynamicClient** are used to work with CR.

**controller-runtime** lib is useful for impl an Operator to manage CR lifecycle.

**[Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) SDK** is dedicated to help you create **CR** and their related **Operators**. It provides cmd to **bootstrap a proj** defining a Manager, and to add resources and their related controllers to the project.

## [Operator SDK](https://sdk.operatorframework.io/)

> A framework to build operators.

It can manage Go/Helm/Ansible projects.

For Go projects, tt provides more functionalities **on top of kubebuilder** to integrate the built operator with the Operator [Lifecycle Manager](https://github.com/operator-framework/operator-lifecycle-manager) and [Operator Hub](https://operatorhub.io/).

## Creating a Project

[Installation](https://book.kubebuilder.io/quick-start.html#installation)

```bash
$ mkdir myresource-kb && cd myresource-kb
# --domain
#		suffix to GVK group, mygroup.myid.com
# --repo
# 		go module name
$ kubebuilder init --domain myid.dev --repo github.com/myid/myresource
```

```bash
# examine cmd avail
$ make help
# build the bin for the Manager
$ make build
# run the Manager locally
$ make run
# or
$ ./bin/manager
```

## Adding a Custom Resource to the Project

For the moment, the Manager does not manage any Controller.

`kubebuilder create api` to add a CR and its related Controller to the project.

```bash
$ kubebuilder create api --group mygroup --version v1alpha1 --kind MyResource
```

```bash
$ tree     
.
├── Dockerfile   # definition of the project, contained the domain and repo as flags to the init command.
├── Makefile
├── PROJECT
├── README.md
├── api          # definition of CR in Go struct, as well as deepcopy-gen
│   └── v1alpha1
│       ├── groupversion_info.go
│       ├── myresource_types.go
│       └── zz_generated.deepcopy.go
├── bin
│   ├── controller-gen
│   ├── controller-gen-v0.16.1
│   └── manager
├── cmd
│   └── main.go
├── config
│   ├── crd
│   │   ├── kustomization.yaml       # kustomize files to build the CRD
│   │   └── kustomizeconfig.yaml
│   ├── default
│   │   ├── kustomization.yaml
│   │   ├── manager_metrics_patch.yaml
│   │   └── metrics_service.yaml
│   ├── manager
│   │   ├── kustomization.yaml
│   │   └── manager.yaml
│   ├── network-policy
│   │   ├── allow-metrics-traffic.yaml
│   │   └── kustomization.yaml
│   ├── prometheus
│   │   ├── kustomization.yaml
│   │   └── monitor.yaml
│   ├── rbac
│   │   ├── kustomization.yaml
│   │   ├── leader_election_role.yaml
│   │   ├── leader_election_role_binding.yaml
│   │   ├── metrics_auth_role.yaml
│   │   ├── metrics_auth_role_binding.yaml
│   │   ├── metrics_reader_role.yaml
│   │   ├── myresource_editor_role.yaml  #  clusterrole for editing/viewing CR
│   │   ├── myresource_viewer_role.yaml
│   │   ├── role.yaml
│   │   ├── role_binding.yaml
│   │   └── service_account.yaml
│   └── samples
│       ├── kustomization.yaml
│       └── mygroup_v1alpha1_myresource.yaml  # CR in YAML
├── go.mod
├── go.sum
├── hack
│   └── boilerplate.go.txt
├── internal
│   └── controller                     # controller for CR
│       ├── myresource_controller.go
│       ├── myresource_controller_test.go
│       └── suite_test.go
└── test
    ├── e2e
    │   ├── e2e_suite_test.go
    │   └── e2e_test.go
    └── utils
        └── utils.go
```

## Building and Deploying Manifests

Builds manifests to be deployed to the cluster.

`config/rbac/role.yaml` used by the Manager, giving accesss to the CR.

`config/crd/bases/mygroup.myid.dev_myresources.yaml` the definition of CR

```bash
$ make manifests
# deploy manifest to cluster
$ make install
```

## Running the Manager Locally

This time, the Manager is handling a Controller, reconciling the MyResource instances.

`internal/controller/myresource_controller.go`

```go
func (r *MyResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// _ = log.FromContext(ctx)

	// TODO(user): your logic here
	logger := log.FromContext(ctx)
	logger.Info("reconcile")

	return ctrl.Result{}, nil
}
```

```bash
# start the Manager locally & use kubeconfig to connect to the cluster with associted permission
$ make run
```

```bash
# new terminal
# create a MyResource instance using the provided sample
# back to previous terminal & check the log
$ kubectl apply -f config/samples/mygroup_v1alpha1_myresource.yaml
```

## Personalizing the Custom Resource

### Editing the Go Structures

`api/v1alpha1/myresource_types.go` to customize CR.

```go
// MyResourceSpec defines the desired state of MyResource
type MyResourceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of MyResource. Edit myresource_types.go to remove/update
	// Foo string `json:"foo,omitempty"`
	Image  string            `json:"image"`
	Memory resource.Quantity `json:"memory"`
}

// MyResourceStatus defines the observed state of MyResource
type MyResourceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	State string `json:"state"`
}
```

### Enabling the Status Subresource

Annotations:

`//+kubebuilder:object:root=true` indicates that MyResource is a Kind. kubebuilder will gen DeepCopyObject() for.

`//+kubebuilder:subresource:status` status subresource must be enabled for MyResource. kubebuilder will add status as subresource in YAML definition of CRD in config/crd/bases.

### Defining Printer Columns

`//+kubebuilder:printcolumn:name="...",type="...",JSONPath="..."`

### Regenerating the Files

```bash
# regenerate the DeepCopy methods
$ make
# regenerate the YAML definition of the CRD with the new fields
$ make manifests
# apply CRD to cluster
$ make install
```

## Implementing the Reconcile Function

Reuse Reconcile fun in previous chapters.

## Adding RBAC Annotations

When the Operator is deployed on the cluster, however, it is running with a specific K8s Service Account and is given **restricted authorizations**.

To help Kubebuilder build this **ClusterRole**, annotations are present in the generated comments of the Reconcile function. **Rules must give full accesss to CR but no others.**

`//+kubebuilder:rbac:groups="...",resources="...",verbs="...;..."`

## Deploying the Operator on the Cluster

Need to build the container image and deploy it to a container image registry.

Create repository named "myresource" on docker.io

```bash
$ make docker-build IMG=yukanyan/myresource:v1alpha1-1
$ make docker-push IMG=yukanyan/myresource:v1alpha1-1
$ make deploy IMG=yukanyan/myresource:v1alpha1-1

# chk
$ kubectl logs deployment/myresource-kb-controller-manager -n myresource-kb-system
```

## Creating a New Version of the Resource

### Defining a New Version

```bash
# create resource only
$ kubebuilder create api --group mygroup --version v1beta1 --kind
```

```go
package v1beta1

// MyResourceSpec defines the desired state of MyResource
type MyResourceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of MyResource. Edit myresource_types.go to remove/update
	// Foo string `json:"foo,omitempty"`
	Image         string            `json:"image"`
    // Memory resource.Quantity `json:"memory"`
	MemoryRequest resource.Quantity `json:"memoryRequest"`
}
```

`// +kubebuilder:storageversion` to declare on which format the resource will be stored in etcd.

Note: here we choose v1alpha1.

### Implementing Hub and Convertible

A Conversion system is provided by the controller-runtime Library.

- The **Hub** interface to mark the version used for the storage
- The **Convertible** interface to provide converters to and from the storage version

Since we choose v1alph1 for the storage, `v1alpha1.MyResource` must impl **Hub** interface.

```go
package v1alpha1

// Hub marks this type as a conversion hub.
func (*MyResource) Hub() {}
```

All other types must impl **Convertible** interface.

```go
package v1beta1

func (src *MyResource) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha1.MyResource)
	dst.Spec.Memory = src.Spec.MemoryRequest
	// Copy other fields
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Image = src.Spec.Image
	dst.Status.State = src.Status.State
	return nil
}

func (dst *MyResource) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha1.MyResource)
	dst.Spec.MemoryRequest = src.Spec.Memory
	// Copy other fields
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Image = src.Spec.Image
	dst.Status.State = src.Status.State
	return nil
}
```

### Setting Up the webhook

`kubebuilder create webhook` cmd to setup. Types:

- **Conversion Webhook** to help convert between resource versions
- **Mutating Admission Webhook** to help set default values on new obj
- **Validating Admission Webhook** to help validate the created or updated objects

```bash
$ kubebuilder create webhook \
	--group mygroup \
	--version v1beta1 \
	--kind MyResource \
	--conversion
```

++ in `main.go`

```go
if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&mygroupv1beta1.MyResource{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "MyResource")
			os.Exit(1)
		}
	}
```

++ in `myresource_webhook.go`

```go
func (r *MyResource) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}
```

### Updating kustomization Files

Enable by webhook by uncommenting [WEBHOOK] and [CERTMANAGER] in:

- `config/default/kustomization.yaml`
- `config/crd/kustomization.yaml`

If you are not using any other webhooks, u also will need to comment on lines:

- `config/webhook/kustomization.yaml`
  - “- manifests.yaml”
- `config/default/kustomization.yaml`
  - “- webhookcainjection_patch.yaml”

You also will need to install [cert-manager](https://cert-manager.io/) to generate certificates based on annotations added by kubebuilder to the CRD.

### Using Various Versions

`config/samples/*`

```yaml
apiVersion: mygroup.myid.dev/v1alpha1
kind: MyResource
metadata:
  labels:
    app.kubernetes.io/name: myresource-kb
    app.kubernetes.io/managed-by: kustomize
  name: myresource-sample
spec:
  # TODO(user): Add fields here
  image: nginx
  memory: 512Mi

```

```yaml
apiVersion: mygroup.myid.dev/v1beta1
kind: MyResource
metadata:
  labels:
    app.kubernetes.io/name: myresource-kb
    app.kubernetes.io/managed-by: kustomize
  name: myresource-sample
spec:
  # TODO(user): Add fields here
  image: nginx
  memoryRequest: 256Mi
```

```bash
$ kubectl apply -f config/samples/mygroup_v1alpha1_myresource.yaml
$ kubectl apply -f config/samples/mygroup_v1beta1_myresource.yaml
```

```bash
$ kubectl get myresources.mygroup.myid.dev -o yaml
$ kubectl get myresources.v1alpha1.mygroup.myid.dev -o yaml
$ kubectl get myresources.v1beta1.mygroup.myid.dev -o yaml
```

