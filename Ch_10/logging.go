package main

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func main() {
	log.SetLogger(zap.New(zap.UseDevMode(true)))
	log.Log.Info("hello world")

	ctrlLog := log.Log.WithValues("package", "controller")
	ctrlLog.Info("hello world")

	// set logger name "controller.main"
	ctrlLog = log.Log.WithName("controller")
	ctrlMainLog := ctrlLog.WithName("main")
	ctrlMainLog.Info("hello world")

}

type MyReconciler struct{}

func (a *MyReconciler) Reconcile(
	ctx context.Context,
	req reconcile.Request,
) (reconcile.Result, error) {
	// get logger from context
	log := log.FromContext(ctx).WithName("reconcile")
	log.Info("reconciling")
	return reconcile.Result{}, nil
}
