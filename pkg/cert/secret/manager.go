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

package secret

import (
	"bytes"
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/szlabs/harbor-cert-injector/api/v1alpha1"
	"github.com/szlabs/harbor-cert-injector/pkg/errs"
	mytypes "github.com/szlabs/harbor-cert-injector/pkg/types"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	namePrefix = "ca-secret"
)

// Manager creates or updates the secret containing the injecting CA data.
type Manager interface {
	// CreateOrUpdate creates or updates the secret containing the CA data.
	// Secret local reference is returned.
	CreateOrUpdate(ctx context.Context, owner *v1alpha1.CertInjection, injection *mytypes.Injection) (corev1.LocalObjectReference, error)

	// AssignOwner set the owner reference of the secret.
	AssignOwner(ctx context.Context, owner *v1alpha1.CertInjection, secret corev1.LocalObjectReference) error
}

// NewManager news a secret creator.
func NewManager(client client.Client, scheme *runtime.Scheme) Manager {
	return &defaultCreator{
		Client: client,
		scheme: scheme,
	}
}

type defaultCreator struct {
	client.Client
	scheme *runtime.Scheme
}

// CreateOrUpdate implements Manager.
func (dc *defaultCreator) CreateOrUpdate(ctx context.Context, owner *v1alpha1.CertInjection, injection *mytypes.Injection) (corev1.LocalObjectReference, error) {
	secretObj, err := dc.get(ctx, types.NamespacedName{
		Namespace: owner.Namespace,
		Name:      secretName(owner.Name),
	})
	if err != nil {
		return corev1.LocalObjectReference{}, err
	}

	// Not existing.
	if secretObj == nil {
		return dc.create(ctx, owner, injection)
	}

	// Has changes?
	savedDNS := secretObj.GetAnnotations()[mytypes.OwnerAnnotationKey]
	if savedDNS != injection.ExternalDNS || !bytes.Equal(secretObj.Data[mytypes.CAKeyInSecret], injection.CACert) {
		return dc.update(ctx, secretObj, injection)
	}

	// No change, return empty name.
	return corev1.LocalObjectReference{}, nil
}

// AssignOwner implements Manager.
func (dc *defaultCreator) AssignOwner(ctx context.Context, owner *v1alpha1.CertInjection, secret corev1.LocalObjectReference) error {
	if owner == nil {
		return errs.New("missing owner to assign")
	}

	if secret.Name == "" {
		return errs.New("missing secret to assign owner")
	}

	// Retrieve secret first.
	obj := &corev1.Secret{}
	if err := dc.Get(ctx, types.NamespacedName{}, obj); err != nil {
		return errs.Wrap("get secret by local ref", err)
	}

	if err := controllerutil.SetControllerReference(owner, obj, dc.scheme); err != nil {
		return errs.Wrap("set controller reference error", err)
	}

	if err := dc.Update(ctx, obj); err != nil {
		return errs.Wrap("update owner reference of secret obj", err)
	}

	return nil
}

func (dc *defaultCreator) get(ctx context.Context, id types.NamespacedName) (*corev1.Secret, error) {
	secretObj := &corev1.Secret{}
	if err := dc.Get(ctx, id, secretObj); err != nil {
		if apierrs.IsNotFound(err) {
			return nil, nil
		}

		return nil, errs.Wrap("get CA secret object error", err)
	}

	return secretObj, nil
}

func (dc *defaultCreator) create(ctx context.Context, owner *v1alpha1.CertInjection, injection *mytypes.Injection) (corev1.LocalObjectReference, error) {
	secretObj := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName(owner.Name),
			Namespace: owner.Namespace,
			Annotations: map[string]string{
				mytypes.OwnerAnnotationKey: injection.ExternalDNS,
			},
		},
		Data: map[string][]byte{
			mytypes.CAKeyInSecret: injection.CACert,
		},
	}

	if err := dc.Create(ctx, secretObj); err != nil {
		return corev1.LocalObjectReference{}, errs.Wrap("create CA secret error", err)
	}

	return corev1.LocalObjectReference{
		Name: secretObj.Name,
	}, nil
}

func (dc *defaultCreator) update(ctx context.Context, secretObj *corev1.Secret, injection *mytypes.Injection) (corev1.LocalObjectReference, error) {
	secretObj.Data[mytypes.CAKeyInSecret] = injection.CACert
	secretObj.SetAnnotations(map[string]string{
		mytypes.OwnerAnnotationKey: injection.ExternalDNS,
	})

	if err := dc.Update(ctx, secretObj); err != nil {
		return corev1.LocalObjectReference{}, errs.Wrap("update CA secret error", err)
	}

	return corev1.LocalObjectReference{
		Name: secretObj.Name,
	}, nil
}

func secretName(ownerName string) string {
	return fmt.Sprintf("%s-%s", namePrefix, ownerName)
}
