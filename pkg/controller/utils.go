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

package controller

import (
	"context"

	"github.com/szlabs/harbor-cert-injector/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/szlabs/harbor-cert-injector/pkg/errs"
)

const (
	onlyWatchResWithLabel = "goharbor.io/cert-injection"
	labelValue            = "enabled"
)

// GVK of an owner object.
type GVK struct {
	APIVersion string
	Kind       string
}

// WithExpectedLabel assure the expected label (with value) is existing.
func WithExpectedLabel(obj client.Object) bool {
	if obj == nil {
		return false
	}

	v, ok := obj.GetLabels()[onlyWatchResWithLabel]
	return ok && v == labelValue
}

// SetupCertInjectionIndex set up index cache for CertInjection.
func SetupCertInjectionIndex(mgr ctrl.Manager, indexKey string, gvk *GVK) error {
	if mgr == nil {
		return errs.New("nil ctrl manager")
	}

	if gvk == nil {
		return errs.New("missing GVK")
	}

	return mgr.GetFieldIndexer().IndexField(context.Background(), &v1alpha1.CertInjection{}, indexKey, func(rawObj client.Object) []string {
		// Grab the object and extract the owner.
		ci := rawObj.(*v1alpha1.CertInjection)

		owner := metav1.GetControllerOf(ci)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != gvk.APIVersion || owner.Kind != gvk.Kind {
			return nil
		}

		// And if so, return it.
		return []string{owner.Name}
	})
}
