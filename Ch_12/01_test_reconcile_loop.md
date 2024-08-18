[ginkgo](https://github.com/onsi/ginkgo) a Testing Framework for Go.

[envtest](https://github.com/kubernetes-sigs/controller-runtime/tree/main/pkg/envtest) package from the controller-runtime Library, which provides a K8s environment for testing.

By default, the package uses local binaries for etcd and kube-apiserver located in `/usr/local/kubebuilder/bin`.

```bash
$ go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
$ sudo ln -s $GOPATH/bin/setup-envtest /usr/local/bin
$ setup-envtest use x.y.z
```

The output of the command will inform you in **which directory** the binaries have been installed.

If you want to use these binaries from the default directory, creaet a symbolic link.

```bash
$ sudo mkdir /usr/local/kubebuilder
$ sudo ln -s /path/to/kubebuilder-envtest/k8s/1.23.5-linux-amd64 /usr/local/kubebuilder/bin
```

or ENV

```bash
$ source <(setup-envtest use -i -p env x.y.z)
$ echo $KUBEBUILDER_ASSETS
```

## Using envtest

The control plane will only run the API Server and etcd, but no controllers = if the operator creates a Deployment, no pod will be created, and the deployment status will never be updated.

```go
import (
    "path/filepath"
    "sigs.k8s.io/controller-runtime/pkg/envtest"
)
testEnv = &envtest.Environment{
    // add CRD by passing the list of directories containing CRD definitions in YAML/JSON.
    // which will be applied to the local cluster when initializing the environment.
    CRDDirectoryPaths: []string{filepath.Join("..", "..", "crd"),},
    // if you want to be altered when the CRD directories do not exist.
    ErrorIfCRDPathMissing: true,
}

// start the env, returns a rest.Config value
// which is the Config value to be used to connect to the local cluster launched by the Environment.
cfg, err := testEnv.Start()

// stop the env
err := testEnv.Stop()
```

Once env is started & rest.Config, we could create the Manager and the Controller then start the Manager like-wise.

## Defining a ginkgo Suite

Use a go test func to start ginkgo specs

```go
import (
    "testing"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestMyReconciler_Reconcile(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite",)
}
```

Then, you can declare BeforeSuite and AfterSuite functions to start/stop the testEnv & Manager.

```go
var (
     testEnv   *envtest.Environment               // used in BeforeSuite and AfterSuite
     ctx       context.Context
     cancel    context.CancelFunc
     k8sClient client.Client                      // client used in tests
)
var _ = BeforeSuite(func() {
     log.SetLogger(zap.New(
          zap.WriteTo(GinkgoWriter),
          zap.UseDevMode(true),
     ))
    
     ctx, cancel = context.WithCancel(             // cancelable ctx
          context.Background(),
     )
    
     testEnv = &envtest.Environment{               // create testEnv
          CRDDirectoryPaths:     []string{
               filepath.Join("..", "..", "crd"),
          },
          ErrorIfCRDPathMissing: true,
     }
     var err error
     // cfg is defined in this file globally.
     cfg, err := testEnv.Start()                   // start testEnv
     Expect(err).NotTo(HaveOccurred())
     Expect(cfg).NotTo(BeNil())
    
     scheme := runtime.NewScheme()                 // build the scheme to pass to the manager
     err = clientgoscheme.AddToScheme(scheme)
     Expect(err).NotTo(HaveOccurred())
     err = mygroupv1alpha1.AddToScheme(scheme)
     Expect(err).NotTo(HaveOccurred())
    
     mgr, err := manager.New(cfg, manager.Options{   // build the manager
          Scheme: scheme,
     })
     Expect(err).ToNot(HaveOccurred())
    
     k8sClient = mgr.GetClient()                     // get client from manager to use for tests
     err = builder.                                  // build controller
          ControllerManagedBy(mgr).
          Named(Name).
          For(&mygroupv1alpha1.MyResource{}).
          Owns(&appsv1.Deployment{}).
          Complete(&MyReconciler{})
     go func() {
          defer GinkgoRecover()
          err = mgr.Start(ctx)                       // start manager from a goroutine
          Expect(err).ToNot(
               HaveOccurred(),
               "failed to run manager",
)
     }()
})
var _ = AfterSuite(func() {
     cancel()                                        // concel ctxt
     err := testEnv.Stop()                           // terminate testEnv
     Expect(err).NotTo(HaveOccurred())
})

```

## Writing the Tests

1. Creating a MyResource instance to verify that the Reconcile function creates the expected “low-level” resources, with the expected definition.
2. Then, when the low-level resources are created, the tests will update the status of the low-level resources to verify that the status of the MyResource instance is updated accordingly.

**Each `It` will be tested separatedly where BeforeEach & AfterEach guard head & tail.**

```go
var _ = Describe("MyResource controller", func() {
    When("creating a MyResource instance", func() {
        BeforeEach(func() {
            // Create the MyResource instance
        })

        AfterEach(func() {
            // Delete the MyResource instance
        })

        It("should create a deployment", func() {
            // Check that the deployment
            // is eventually created
        })

        When("deployment is found", func() {
            BeforeEach(func() {
                // Wait for the deployment
                // to be eventually created
            })

            It("should be owned by the MyResource instance", func() {
                // Check ownerReference in Deployment
                // references the MyResource instance
            })

            It("should use the image specified in MyResource instance", func() {
            })

            When("deployment ReadyReplicas is 1", func() {
                BeforeEach(func() {
                    // Update the Deployment status
                    // to ReadyReplicas=1
                })

                It("should set status ready for MyResource instance", func() {
                    // Check the status of MyResource instance
                    // is eventually Ready
                })
            })
        })
    })
})
```

**Test 1**

- **When**: creating a MyResource instance
- **Before**: // Create the MyResource instance
- **It**: should create a deployment
- **After**: // Delete the MyResource instance

**Test 2**

- **When**: creating a MyResource instance
- **Before**: // Create the MyResource instance
- **When**: deployment is found
- **Before**: // Wait deployment eventually created
- **It**: should be owned by the MyResource instance
- **After**: // Delete the MyResource instance

**Test 3**

- **When**: creating a MyResource instance
- **Before**: // Create the MyResource instance
- **When**: deployment is found
- **Before**: // Wait deployment eventually created
- **It**: should use image specified in MyResource instance
- **After**: // Delete the MyResource instance

**Test 4**

- **When**: creating a MyResource instance
- **Before**: // Create the MyResource instance
- **When**: deployment is found
- **Before**: // Wait deployment eventually created
- **When**: deployment ReadyReplicas is 1
- **Before**: // Update Deployment status to ReadyReplicas=1
- **It**: should set status ready for MyResource instance
- **After**: // Delete the MyResource instance

