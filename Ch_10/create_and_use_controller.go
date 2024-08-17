package main

import (
	"context"
	"fmt"
	mygroupv1alpha1 "github.com/myid/myresource-crd/pkg/apis/mygroup.example.com/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func main() {
	// 1. create scheme with native resources & CR MyResource
	scheme := runtime.NewScheme()
	err := clientgoscheme.AddToScheme(scheme)
	if err != nil {
		return
	}
	err = mygroupv1alpha1.AddToScheme(scheme)
	if err != nil {
		return
	}

	// 2. create manager using the scheme
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Scheme: scheme,
	})
	if err != nil {
		panic(err)
	}

	//// 3. create controller, attached to manager, passing a Reconciler impl
	//ctrl, err := controller.New("my-operator", mgr, controller.Options{
	//	Reconciler: &MyReconciler1{},
	//})
	//
	//// 4. start watching MyResource instances as a primary resource
	//err = ctrl.Watch(&source.Kind{
	//	Type: &mygroupv1alpha1.MyResource{},
	//}, &handler.EnqueueRequestForObject{})
	//
	//// 5. start watching Pod instances as an owned resource
	//err = ctrl.Watch(&source.Kind{
	//	Type: &corev1.Pod{},
	//}, &handler.EnqueueRequestForOwner{
	//	OwnerType:    &corev1.Pod{},
	//	IsController: true,
	//})

	// Preferred: create controller by builder
	err = builder.ControllerManagedBy(mgr).
		For(&mygroupv1alpha1.MyResource{}).
		Owns(&corev1.Pod{}).
		//Complete(&MyReconciler1{})
		//Complete(&MyReconciler2{})
		Complete(&MyReconciler3{})

	// 6. start the manager
	err = mgr.Start(context.Background())
	if err != nil {
		panic(err)
	}
}

type MyReconciler1 struct{}

// Reconcile Implementation of the Reconcile method, display the namespace & name of the instance to reconcile
func (o *MyReconciler1) Reconcile(ctx context.Context, r reconcile.Request) (reconcile.Result, error) {
	fmt.Printf("reconcile %v\n", r)
	return reconcile.Result{}, nil
}

type MyReconciler2 struct {
	client client.Client
	cache  cache.Cache
	scheme *runtime.Scheme
}

func (o *MyReconciler2) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	fmt.Printf("reconcile %v\n", req)
	fmt.Printf("client %v\n", o.client)
	fmt.Printf("cache %v\n", o.cache)
	fmt.Printf("scheme %v\n", o.scheme)
	return reconcile.Result{}, nil
}

// The Reconciler implementations need to implement the specific injector interfaces from the inject package

func (o *MyReconciler2) InjectClient(
	c client.Client,
) error {
	o.client = c
	return nil
}

func (o *MyReconciler2) InjectCache(
	c cache.Cache,
) error {
	o.cache = c
	return nil
}

func (o *MyReconciler2) InjectScheme(
	s *runtime.Scheme,
) error {
	o.scheme = s
	return nil
}

// Use client

type MyReconciler3 struct {
	client client.Client
}

func (r *MyReconciler3) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	fmt.Printf("reconcile %v\n", req)

	// get info about a resource
	myres := mygroupv1alpha1.MyResource{}
	err := r.client.Get(ctx, req.NamespacedName, &myres)
	if err != nil {
		return reconcile.Result{}, err
	}
	err = r.client.Get(ctx, req.NamespacedName, &myres, &client.GetOptions{
		Raw: &v1.GetOptions{
			ResourceVersion: "0",
		},
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	// list resources
	myreslist := mygroupv1alpha1.MyResourceList{}
	err = r.client.List(ctx, &myreslist, &client.ListOptions{}, client.InNamespace(req.Namespace))
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, res := range myreslist.Items {
		fmt.Printf("res %v\n", res.GetName())
	}

	// create a resource
	podToCreate := corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      "my-pod",
			Namespace: "my-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "main",
					Image: "nginx",
				},
			},
		},
	}
	err = r.client.Create(ctx, &podToCreate)
	if err != nil {
		return reconcile.Result{}, err
	}

	// delete a resource
	podToDelete := corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      "my-pod",
			Namespace: "my-namespace",
		},
	}
	err = r.client.Delete(ctx, &podToDelete)
	if err != nil {
		return reconcile.Result{}, err
	}

	// delete a collection of resources
	err = r.client.DeleteAllOf(ctx, &mygroupv1alpha1.MyResource{}, client.InNamespace(req.Namespace))
	if err != nil {
		return reconcile.Result{}, err
	}

	// patch a resource
	// 1. server-side apply
	deployToApply := appsv1.Deployment{
		TypeMeta: v1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "my-deploy",
			Namespace: "my-namespace",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "app1",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"app": "app1",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "main",
							Image: "nginx",
						},
					},
				},
			},
		},
	}
	err = r.client.Patch(ctx, &deployToApply, client.Apply, client.FieldOwner("my-controller"), client.ForceOwnership)
	if err != nil {
		return reconcile.Result{}, err
	}

	// 2. strategic merge patch - add an env to the 1st container of a deployment pod template
	var deploymentRead appsv1.Deployment
	err = r.client.Get(ctx, req.NamespacedName, &deploymentRead)
	if err != nil {
		return reconcile.Result{}, err
	}

	patch1 := client.StrategicMergeFrom(deploymentRead.DeepCopy())

	deploymentModified := deploymentRead.DeepCopy()
	deploymentModified.Spec.Template.Spec.Containers[0].Env = append(deploymentModified.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  "MY_ENV",
		Value: "MY_VALUE",
	})

	err = r.client.Patch(ctx, deploymentModified, patch1)

	// 3. merge patch
	patch2 := client.MergeFrom(deploymentRead.DeepCopy())
	deploymentModified = deploymentRead.DeepCopy()
	deploymentModified.Spec.Replicas = new(int32)
	*deploymentModified.Spec.Replicas = 3
	err = r.client.Patch(ctx, deploymentModified, patch2)

	// update the status
	myres.Status.State = "done"
	err = r.client.Status().Update(ctx, &myres)
	if err != nil {
		return reconcile.Result{}, err
	}

	// patch the status
	patch := client.MergeFrom(deploymentRead.DeepCopy())
	myres.Status.State = "done"
	err = r.client.Status().Patch(ctx, &myres, patch)

	return reconcile.Result{}, nil
}

func (r *MyReconciler3) InjectClient(
	c client.Client,
) error {
	r.client = c
	return nil
}
