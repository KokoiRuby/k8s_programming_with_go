K8s API is organized in **groups**, each group contains one or more **verionsed** resources.

To work with K8s API: **API Library (Def.) & API Machinery (Tool to talk)** & **Client-go (Access to)**.

K8s API is extensible through **CRD** CustomResourceDefinition.

```go
import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)
```

## Performing Operations in Go

Use **clientset** to perform op on CRD.

```go
import (
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
)

clientset, err := clientset.NewForConfig(config)
if err != nil {
	return err
}
ctx := context.Background()
list, err := clientset.ApiextensionsV1().
                CustomResourceDefinitions().
                List(ctx, metav1.ListOptions{})
```

## The CustomResourceDefinition in Detail

```go
type CustomResourceDefinition struct {
    metav1.TypeMeta
    metav1.ObjectMeta
    
    Spec CustomResourceDefinitionSpec
    Status CustomResourceDefinitionStatus
}

type CustomResourceDefinitionSpec struct {
    Group string
    Scope ResourceScope                   // ClusterScoped or NamespaceScoped.
    
    Names CustomResourceDefinitionNames
    Versions []CustomResourceDefinitionVersion
    Conversion *CustomResourceConversion
    PreserveUnknownFields bool
}
```

**Naming**

```go
type CustomResourceDefinitionNames struct {
    Plural       string    // pods
    Singular     string    // pod, lowercase singular
    ShortNames []string    // po, lowercase short names
    Kind         string    // Pod, CamelCase
    ListKind     string    // PodList, used during the serialization of lists of this resource
    Categories []string    // a list of grouped resources the resource belongs to
}
```

**DefinitionVersion**

```go
type CustomResourceDefinitionVersion struct {
    Name                      string  // v<number>[(alpha|beta)<number>]
    Served                    bool    // whether this specific ver must be served by the API Server.
    Storage                   bool    // indicates whether this specific ver is the one used for persisting.
    Deprecated                bool    // indicates whether this specific ver of the resource is deprecated.
    DeprecationWarning        *string // msg returned when Deprecated is true
    Schema                    *CustomResourceValidation       // validate the data sent to the API
    Subresources              *CustomResourceSubresources     // subresources that will be served for
    AdditionalPrinterColumns []CustomResourceColumnDefinition
}

type CustomResourceSubresources struct {
	Status *CustomResourceSubresourceStatus  // /status will be served
	Scale  *CustomResourceSubresourceScale   // /scale will be seved
}
```

**Converting btw versions.**

```go
type CustomResourceConversion struct {
    Strategy ConversionStrategyType  // NoneConverter or WebhookConverter.
    Webhook  *WebhookConversion      // API Server is to call an external webhook to do the conversion.
}
```

## Schema of the Resource

Data Types: `string` (date/date-time/byte/int-or-string), `number`(float/double/int32/int64), `integer`, `boolean`, `array`, object. 

Property: required, enum, additionalProperties (map)

```yaml
schema:
  openAPIV3Schema:
    type: object
    properties:
      # fields in Spec: image/replicas/port
      spec:
        type: object
        properties:
          image:
            type: string
          replicas:
            type: integer
          port:
            type: string
            format: int-or-string
        required:
          - image
          - replicas
      # fields in Status: state
      status:
        type: object
        properties:
          state:
            type: string
            enum:
              - waiting
              - running
```

## Deploying a Custom Resource Definition

Define & Create a CRD.

```yaml

❽ Short names of the new resource, you can use kubectl get my,
kubectl get myres
❾ Adds the resource to the category all; resources of this kind will
appear when running kubectl get all
❿ v1alpha1 version is the only version defined for the new
resource
⓫ Defines the new resource schema as an object, with no field

apiVersion: apiextensions.k8s.io/v1     # 1. The group and version of the CRD resource
kind: CustomResourceDefinition          # 2. The kind of the CRD resource
metadata:
  name: myresources.mygroup.example.com # 3. The complete name of the new resource, including its group
spec:
  group: mygroup.example.com            # 4. The group the new resource belongs to
  scope: Namespaced                     # 5. The new resource can be created in specific namespaces
  names:
    plural: myresources                 # 6. The plural name of the new resource
    singular: myresource                # 7. The singular name of the resource
    shortNames:                         # 8. Short names of the new resource
      - my
      - myres
  kind: MyResource             
  categories:                  # 9. resources of this kind will appear when running kubectl get all
    - all
  versions:
    - name: v1alpha1           # 10. v1alpha1 version is the only version defined for the new resource
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object         # 11. Defines the new resource schema as an object, with no field
```

```bash
$ kubectl apply -f myresource.yaml
# GVR
$ curl http://localhost:8001/apis/mygroup.example.com/v1alpha1/myresources
$ curl http://localhost:8001/apis/mygroup.example.com/v1alpha1/namespaces/default/myresources
$ kubectl get myresources
```

```bash
$ kubectl apply -f - <<EOF
apiVersion: mygroup.example.com/v1alpha1
kind: MyResource
metadata:
	name: myres1
EOF

$ kubectl get myresources
```

## Additional Printer Columns

The **AdditionalPrinterColumns** field of the CRD spec is used to indicate which columns of the resource you want to be displayed in the output of `kubectl get <resource>`. 

**name** and **age** are returned by default if not specified.

For each additional column, you need to specify a **name**, a **JSON path**. and a type.

- The **name** will be used as a header for the column in the output.
- The **JSON** path is used by the API Server to get the value for this column from the resource data.
- The **type** is an indication for kubectl to display this data.

```yaml
# ...
schema:
  openAPIV3Schema:
    type: object
    properties:
      spec:
        type: object
        properties:
          image:
            type: string
          memory:
            x-kubernetes-int-or-string: true
      status:
        type: object
        properties:
          state:
            type: string
  additionalPrinterColumns:
     - name: image
       jsonPath: .spec.image
       type: string
     - name: memory
       jsonPath: .spec.memory
       type: string
     - name: age
       jsonPath: .metadata.creationTimestamp
       type: date
```

```bash
$ kubectl apply -f myresource.yaml

$ cat > myres1.yaml <<EOF
apiVersion: mygroup.example.com/v1alpha1
kind: MyResource
metadata:
	name: myres1
spec:
	image: nginx 
	memory: 1024Mi
EOF

$ kubectl apply -f myres1.yaml
```



