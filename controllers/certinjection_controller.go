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
	"fmt"

	"github.com/szlabs/harbor-cert-injector/api/v1alpha1"
	"github.com/szlabs/harbor-cert-injector/pkg/cert/injector"
	"github.com/szlabs/harbor-cert-injector/pkg/controller"
	mytypes "github.com/szlabs/harbor-cert-injector/pkg/types"

	appv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	indexKey = ".metadata.controller"
)

// CertInjectionReconciler reconciles a CertInjection object
type CertInjectionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=day2-operations.goharbor.io,resources=certinjections,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=day2-operations.goharbor.io,resources=certinjections/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=day2-operations.goharbor.io,resources=certinjections/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *CertInjectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.WithValues("cert injection", req.NamespacedName)

	certInjection := &v1alpha1.CertInjection{}
	if err := r.Get(ctx, req.NamespacedName, certInjection); err != nil {
		logger.Error(err, "unable to fetch cert injection")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !certInjection.GetObjectMeta().GetDeletionTimestamp().IsZero() {
		logger.Info("object is being deleted")
		return ctrl.Result{}, nil
	}

	// Check the existence of the underlying daemon set.
	dsList := &appv1.DaemonSetList{}
	if err := r.List(ctx, dsList, client.InNamespace(req.Namespace), client.MatchingFields{indexKey: req.Name}); err != nil {
		return ctrl.Result{}, err
	}

	ijp := injector.NewDaemonSetProvider(r.Client, r.Scheme)
	if len(dsList.Items) == 0 {
		logger.Info("underlying daemonset not found")

		// Not found.
		if err := ijp.Inject(ctx, certInjection); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		ds := &dsList.Items[0]
		oldInjectionV := ds.GetAnnotations()[mytypes.InjectionVersionAnnotationKey]
		newInjectionV := certInjection.GetResourceVersion()

		// If update is needed.
		if oldInjectionV != newInjectionV {
			logger.Info("resource version changes found", "old", oldInjectionV, "new", newInjectionV)

			updatedDs := ijp.DesiredInjector(certInjection)
			ds.Spec = *updatedDs.Spec.DeepCopy()
			ds.Annotations[mytypes.InjectionVersionAnnotationKey] = newInjectionV

			if err := r.Update(ctx, ds); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	logger.Info("reconcile completed")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CertInjectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &appv1.DaemonSet{}, indexKey, func(rawObj client.Object) []string {
		// Grab the object and extract the owner.
		ds := rawObj.(*appv1.DaemonSet)

		owner := metav1.GetControllerOf(ds)
		if owner == nil {
			return nil
		}

		// Make sure it's daemon set.
		if owner.APIVersion != appv1.SchemeGroupVersion.String() || owner.Kind != "DaemonSet" {
			return nil
		}

		// And if so, return it.
		return []string{owner.Name}

	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CertInjection{}).
		Owns(&appv1.DaemonSet{}).
		Complete(r)
}

func nameDS(nsn types.NamespacedName) types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("cjds-%s", nsn.Name),
		Namespace: nsn.Namespace,
	}
}

func init() {
	controller.AddToControllerList(&CertInjectionReconciler{})
}
