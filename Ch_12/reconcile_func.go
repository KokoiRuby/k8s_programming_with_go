package main

import (
	"context"
	"fmt"
	mygroupv1alpha1 "github.com/myid/myresource-crd/pkg/apis/mygroup.example.com/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func main() {
	log.SetLogger(zap.New(zap.UseDevMode(true)))

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
		})
	if err != nil {
		return
	}

	eventRecorder := mgr.GetEventRecorderFor("MyResource")

	// build controller & start
	err = builder.ControllerManagedBy(mgr).
		For(&mygroupv1alpha1.MyResource{}).
		Owns(&appsv1.Deployment{}).
		Complete(&MyReconciler{
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

type MyReconciler struct {
	client        client.Client
	EventRecorder record.EventRecorder
}

func (r *MyReconciler) InjectClient(c client.Client) error {
	r.client = c
	return nil
}

func (r *MyReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	reconcileLogger := log.FromContext(ctx)
	reconcileLogger.Info("Reconciling MyResource, trying to get myresource instance")

	// 1. Get the definition of the resource to reconcile
	myRes := &mygroupv1alpha1.MyResource{}
	err := r.client.Get(ctx, req.NamespacedName, myRes, &client.GetOptions{})
	if err != nil {
		// 2. If resource does not exist, return immediately
		if errors.IsNotFound(err) {
			reconcileLogger.Info("MyResource not found")
			return reconcile.Result{}, nil
		}
	}

	// 3. Build the ownerReference pointing to the resource to reconcile
	ownerReferences := metav1.NewControllerRef(myRes, mygroupv1alpha1.SchemeGroupVersion.WithKind("MyResource"))

	// 4. Use Server-side Apply for the “low-level” deployment
	err = r.applyDeployment(ctx, myRes, ownerReferences)
	if err != nil {
		return reconcile.Result{}, err
	}

	// 5. Compute the status of the resource based on the “low-level” deployment
	status, err := r.computeStatus(ctx, myRes)
	if err != nil {
		return reconcile.Result{}, err
	}
	// 6. Update the status of the resource to reconcile
	myRes.Status = *status
	reconcileLogger.Info("updating status", "status", myRes.Status)
	err = r.client.Status().Update(ctx, myRes)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *MyReconciler) applyDeployment(ctx context.Context, myRes *mygroupv1alpha1.MyResource, ownerRef *metav1.OwnerReference) error {
	deploy := createDeployment(myRes, ownerRef)
	// 7. Server-side Apply
	err := r.client.Patch(ctx, deploy, client.Apply, client.FieldOwner("MyResourceReconciler"), client.ForceOwnership)
	return err
}

func createDeployment(myres *mygroupv1alpha1.MyResource, ownerRef *metav1.OwnerReference) *appsv1.Deployment {
	deploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      myres.GetName() + "-deployment",
			Namespace: myres.GetNamespace(),
			Labels: map[string]string{
				"myresource": myres.GetName(),
			},
			// 8. Set the OwnerReference to point to the resource to reconcile
			OwnerReferences: []metav1.OwnerReference{*ownerRef},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"myresource": myres.GetName(),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"myresource": myres.GetName(),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "main",
							// 9. Use the Image defined in the resource to reconcile
							Image: myres.Spec.Image,
							// 10. Use the Memory defined in the resource to reconcile
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: myres.Spec.Memory,
								},
							},
						},
					},
				},
			},
		},
	}
	return deploy
}

func (r *MyReconciler) computeStatus(ctx context.Context, myRes *mygroupv1alpha1.MyResource) (*mygroupv1alpha1.MyResourceStatus, error) {
	statusLogger := log.FromContext(ctx)
	result := mygroupv1alpha1.MyResourceStatus{
		State: "Building",
	}

	// 11. Get the deployment created for this resource to reconcile
	deployList := appsv1.DeploymentList{}
	err := r.client.List(ctx, &deployList, client.InNamespace(myRes.GetNamespace()), client.MatchingLabels{
		"myresource": myRes.GetName(),
	})
	if err != nil {
		return nil, err
	}

	if len(deployList.Items) == 0 {
		statusLogger.Info("No Deployments found")
		return &result, nil
	}

	if len(deployList.Items) > 1 {
		statusLogger.Info("Multiple Deployments found", "count", len(deployList.Items))
		return nil, fmt.Errorf("%v deployment found, expected 1", len(deployList.Items))
	}

	// 12. Get the status of the unique Deployment found
	status := deployList.Items[0].Status
	statusLogger.Info("Got Deployment status", "status", status)

	// 13. When replicas is 1, set status Ready for the reconciled resource
	if status.ReadyReplicas == 1 {
		result.State = "Ready"
	}

	return &result, nil
}
