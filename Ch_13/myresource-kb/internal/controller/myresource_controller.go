/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mygroupv1alpha1 "github.com/myid/myresource/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MyResourceReconciler reconciles a MyResource object
type MyResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=mygroup.myid.dev,resources=myresources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mygroup.myid.dev,resources=myresources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mygroup.myid.dev,resources=myresources/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyResource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *MyResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// _ = log.FromContext(ctx)

	// TODO(user): your logic here
	logger := log.FromContext(ctx)
	logger.Info("getting myresource instance")

	// reuse
	myRes := mygroupv1alpha1.MyResource{}
	err := r.Client.Get(
		ctx,
		req.NamespacedName,
		&myRes,
		&client.GetOptions{},
	)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("resource is not found")
			return reconcile.Result{}, nil
		}
	}

	ownerRef := metav1.NewControllerRef(&myRes, mygroupv1alpha1.GroupVersion.WithKind("MyResource"))

	err = r.applyDeployment(ctx, &myRes, ownerRef)
	if err != nil {
		return reconcile.Result{}, err
	}

	status, err := r.computeStatus(ctx, &myRes)
	if err != nil {
		return reconcile.Result{}, err
	}

	myRes.Status = *status
	logger.Info("updating status", "state", status.State)
	err = r.Client.Status().Update(ctx, &myRes)
	if err != nil {
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mygroupv1alpha1.MyResource{}).
		Complete(r)
}
