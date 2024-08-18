package main

import (
	"context"
	mygroupv1alpha1 "github.com/myid/myresource-crd/pkg/apis/mygroup.example.com/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"testing"
)

func TestMyReconciler_Reconcile(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var (
	// used in BeforeSuite and AfterSuite
	testEnv *envtest.Environment
	ctx     context.Context
	cancel  context.CancelFunc
	// client used in tests
	k8sClient client.Client
)

var _ = BeforeSuite(func() {
	log.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	// cancelable ctx
	ctx, cancel = context.WithCancel(context.Background())

	// create testEnv
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join(".")},
		ErrorIfCRDPathMissing: true,
	}

	// start testEnv
	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// build the scheme to pass to the manager
	scheme := runtime.NewScheme()
	err = clientgoscheme.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	err = mygroupv1alpha1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	mgr, err := manager.New(cfg, manager.Options{
		Scheme: scheme,
	})
	Expect(err).NotTo(HaveOccurred())

	// get client from manager to use for tests
	k8sClient = mgr.GetClient()
	// build controller
	err = builder.ControllerManagedBy(mgr).
		Named("Name").
		For(&mygroupv1alpha1.MyResource{}).
		Owns(&appsv1.Deployment{}).
		Complete(&MyReconciler{})

	// start manager from a goroutine
	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).ToNot(
			HaveOccurred(),
			"failed to run manager",
		)
	}()
})

var _ = AfterSuite(func() {
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
