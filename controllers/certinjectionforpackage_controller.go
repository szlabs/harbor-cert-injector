/*
Copyright 2022 szou.

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

package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/szlabs/harbor-cert-injector/api/v1alpha1"
	"github.com/szlabs/harbor-cert-injector/pkg/cert/injection"
	"github.com/szlabs/harbor-cert-injector/pkg/controller"
	"github.com/szlabs/harbor-cert-injector/pkg/errs"
	packagev1alpha1 "github.com/vmware-tanzu/carvel-kapp-controller/pkg/apis/packaging/v1alpha1"
)

// CertInjectionForPackageReconciler reconciles a kapp PackageInstall object
type CertInjectionForPackageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=packaging.carvel.dev,resources=packageinstalls,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *CertInjectionForPackageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("package install", req.NamespacedName)

	// Init the common reconciler.
	reconciler := injection.NewBuilder().
		UseClient(r.Client).
		WithLogger(logger).
		WithScheme(r.Scheme).
		Reconciler()

	// Do reconcile.
	if err := reconciler.Reconcile(ctx, req.NamespacedName, func() client.Object {
		return &packagev1alpha1.PackageInstall{}
	}); err != nil {
		if !errs.IsTLSNotEnabledError(err) {
			return ctrl.Result{}, err
		}

		logger.Info("Skip reconcile", "cause", err)
	}

	logger.Info("Reconcile loop completed")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CertInjectionForPackageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()

	return ctrl.NewControllerManagedBy(mgr).
		For(&packagev1alpha1.PackageInstall{}, controller.WithExpectedLabelPredicates()).
		Owns(&v1alpha1.CertInjection{}).
		Complete(r)
}

func init() {
	controller.AddToControllerList(&CertInjectionForPackageReconciler{})
}
