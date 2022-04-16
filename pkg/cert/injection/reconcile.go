// Copyright Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package injection

import (
	"context"
	"fmt"

	"github.com/szlabs/harbor-cert-injector/pkg/controller"

	"github.com/go-logr/logr"
	"github.com/szlabs/harbor-cert-injector/api/v1alpha1"
	"github.com/szlabs/harbor-cert-injector/pkg/cert/extractor"
	"github.com/szlabs/harbor-cert-injector/pkg/cert/secret"
	"github.com/szlabs/harbor-cert-injector/pkg/errs"
	mytypes "github.com/szlabs/harbor-cert-injector/pkg/types"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	caInjectionNamePrefix = "ca-injection"
)

// ObjectFactoryFunc for newing a typed object
type ObjectFactoryFunc func() client.Object

// Reconciler is for doing the common cert injection reconciling logic.
type Reconciler interface {
	// Reconcile the object and make sure a corresponding v1alpha1.CertInjection is there.
	// The object might be:
	//  - HarborCluster CR
	//  - PackageInstall CR
	//  - Secret (with expected label)
	Reconcile(ctx context.Context, name types.NamespacedName, objFacFunc ObjectFactoryFunc) error
}

// ReconcilerBuilder is used to build a usable Reconciler.
type ReconcilerBuilder interface {
	// WithScheme sets scheme.
	WithScheme(scheme *runtime.Scheme) ReconcilerBuilder
	// WithLogger sets logger.
	WithLogger(logger logr.Logger) ReconcilerBuilder
	// UseClient sets client.
	UseClient(client client.Client) ReconcilerBuilder
	// Reconciler returns the ready Reconciler.
	Reconciler() Reconciler
}

type commonController struct {
	client.Client
	scheme    *runtime.Scheme
	logger    logr.Logger
	secretMgr secret.Manager
}

// NewBuilder news a common reconciler builder.
func NewBuilder() ReconcilerBuilder {
	return &commonController{}
}

// Reconcile implements Reconciler.
func (cc *commonController) Reconcile(ctx context.Context, name types.NamespacedName, objFacFunc ObjectFactoryFunc) error {
	if err := cc.validate(); err != nil {
		return errs.Wrap("common reconciler", err)
	}

	if objFacFunc == nil {
		return errs.New("nil object factory func")
	}

	// Get the target by the provided name and marshal into the object created by the factory func.
	target := objFacFunc()
	if err := cc.Get(ctx, name, target); err != nil {
		if apierrs.IsNotFound(err) {
			cc.logger.Error(err, "failed to get the resource")
			return nil
		}

		return errs.Wrap("failed to get target resource", err)
	}

	// Resource is being deleted.
	if !target.GetDeletionTimestamp().IsZero() {
		return errs.New("target resource is being deleted")
	}

	// Extract cert injection data from the target object for latter usage.
	GVK := target.GetObjectKind().GroupVersionKind()
	injection, err := extractor.Providers(cc.Client).Get(GVK.String()).Extract(ctx, target)
	if err != nil {
		return errs.Wrap("extract cert data error", err)
	}

	// Check if there has already been an underlying owning cert injection CR.
	var ciList v1alpha1.CertInjectionList
	if err := cc.List(ctx, &ciList, client.InNamespace(name.Namespace), client.MatchingLabels{
		mytypes.OwnerGVKLabel:  controller.FormatGVKToLabelValue(GVK),
		mytypes.OwnerNameLabel: target.GetName(),
	}); err != nil {
		return errs.Wrap("unable to list underlying cert injections", err)
	}

	var certInjection *v1alpha1.CertInjection
	if len(ciList.Items) != 0 {
		certInjection = &ciList.Items[0]
	} else {
		cc.logger.Info("Create new as underlying cert injection not found")
		// Not found and create a new CR.
		cij, err := cc.createCertInjectionCR(target)
		if err != nil {
			return errs.Wrap("failed to create CertInjection CR", err)
		}

		certInjection = cij
	}

	// Create or update the CA secret first.
	// If injection has no changes, no changes will be applied to the existing secret.
	secretRef, err := cc.secretMgr.CreateOrUpdate(ctx, certInjection, injection)
	if err != nil {
		return errs.Wrap("CA secret manager error", err)
	}

	cc.logger.V(5).Info("Handle CA secret", "secret reference", secretRef)

	// Secret has been changed (created or updated).
	if secretRef.Name != "" {
		isCreate := certInjection.Spec.ExternalDNS == ""

		cc.logger.Info("Source CA secret has changes", "isCreate", isCreate)

		// Set the spec.
		certInjection.Spec.ExternalDNS = injection.ExternalDNS
		certInjection.Spec.CertSecret = secretRef
		// Update the last-update timestamp.
		certInjection.Annotations[mytypes.LastUpdateTimestampAnnotationKey] = fmt.Sprintf("%s", metav1.NowMicro())

		// Update the status condition.
		for _, c := range certInjection.Status.Conditions {
			cp := &c
			if cp.Type == mytypes.ConditionCAReady && cp.Status == corev1.ConditionFalse {
				cp.Status = corev1.ConditionTrue
				break
			}
		}

		// Need to create CertInjection CR.
		if isCreate {
			if err := cc.Create(ctx, certInjection); err != nil {
				return errs.Wrap("failed to create cert injection CR", err)
			}

			cc.logger.Info("Cert injection is created", "name", certInjection.GetName())

			// Set owner reference to the created CA secret.
			if err := cc.secretMgr.AssignOwner(ctx, certInjection, secretRef); err != nil {
				return errs.Wrap("failed to assign owner to secret", err)
			}

			return nil
		}

		// Need to update the CertInjection CR.
		if err := cc.Update(ctx, certInjection); err != nil {
			return errs.Wrap("update cert injection CR error", err)
		}

		cc.logger.Info("Cert injection is updated", "name", certInjection.GetName())
	}

	return nil
}

// WithScheme implements ReconcilerBuilder.
func (cc *commonController) WithScheme(scheme *runtime.Scheme) ReconcilerBuilder {
	if scheme != nil {
		cc.scheme = scheme
	}

	return cc
}

// WithLogger implements ReconcilerBuilder.
func (cc *commonController) WithLogger(logger logr.Logger) ReconcilerBuilder {
	cc.logger = logger
	return cc
}

// UseClient implements ReconcilerBuilder.
func (cc *commonController) UseClient(client client.Client) ReconcilerBuilder {
	if client != nil {
		cc.Client = client
	}

	return cc
}

// Reconciler implements ReconcilerBuilder.
func (cc *commonController) Reconciler() Reconciler {
	if cc.secretMgr == nil {
		if cc.Client != nil && cc.scheme != nil {
			cc.secretMgr = secret.NewManager(cc.Client, cc.scheme)
		}
	}

	return cc
}

func (cc *commonController) validate() error {
	if cc.Client == nil {
		return errs.New("missing client")
	}

	if cc.scheme == nil {
		return errs.New("missing scheme")
	}

	if cc.secretMgr == nil {
		return errs.New("missing secret manager")
	}

	return nil
}

func (cc *commonController) createCertInjectionCR(target client.Object) (*v1alpha1.CertInjection, error) {
	targetREF, err := reference.GetReference(cc.scheme, target)
	if err != nil {
		return nil, errs.Wrap("get object reference error", err)
	}

	return &v1alpha1.CertInjection{
		TypeMeta: metav1.TypeMeta{
			Kind:       mytypes.CertInjection,
			APIVersion: v1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: targetREF.Namespace,
			Name:      caInjectionName(targetREF.Name),
			Annotations: map[string]string{
				mytypes.LastUpdateTimestampAnnotationKey: metav1.NowMicro().String(),
			},
			Labels: map[string]string{
				mytypes.OwnerGVKLabel:  controller.FormatGVKToLabelValue(target.GetObjectKind().GroupVersionKind()),
				mytypes.OwnerNameLabel: target.GetName(),
			},
		},
		Spec: v1alpha1.CertInjectionSpec{},
		Status: v1alpha1.CertInjectionStatus{
			CertSourceRef: targetREF,
			Conditions: []v1alpha1.CertInjectionCondition{
				{
					Type:   mytypes.ConditionCAReady,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}, nil
}

func caInjectionName(name string) string {
	return fmt.Sprintf("%s-%s", caInjectionNamePrefix, name)
}
