package main

import (
	"context"
	"fmt"
	mygroupv1alpha1 "github.com/myid/myresource-crd/pkg/apis/mygroup.example.com/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func main() {
	scheme := runtime.NewScheme()
	err := clientgoscheme.AddToScheme(scheme)
	if err != nil {
		return
	}
	err = mygroupv1alpha1.AddToScheme(scheme)
	if err != nil {
		return
	}

	mgr, err := manager.New(
		config.GetConfigOrDie(),
		manager.Options{
			Scheme: scheme,
		},
	)
	if err != nil {
		return
	}

	eventRecorder := mgr.GetEventRecorderFor("MyResource")

	err = builder.
		ControllerManagedBy(mgr).
		For(&mygroupv1alpha1.MyResource{}).
		Owns(&corev1.Pod{}).
		Complete(&MyReconciler4{
			EventRecorder: eventRecorder,
		})
	if err != nil {
		return
	}

	err = mgr.Start(context.Background())
	if err != nil {
		return
	}
}

type MyReconciler4 struct {
	client client.Client
	// embed
	EventRecorder record.EventRecorder
}

func (a *MyReconciler4) Reconcile(
	ctx context.Context,
	req reconcile.Request,
) (reconcile.Result, error) {
	fmt.Printf("reconcile %v\n", req)

	myres := mygroupv1alpha1.MyResource{}
	err := a.client.Get(
		ctx,
		req.NamespacedName,
		&myres,
	)
	if err != nil {
		fmt.Printf("%v\n", err)
		return reconcile.Result{}, err
	}

	a.EventRecorder.Event(&myres, corev1.EventTypeNormal, "Reconcile", "reconciling")

	return reconcile.Result{}, nil
}

func (r *MyReconciler4) InjectClient(
	c client.Client,
) error {
	r.client = c
	return nil
}
