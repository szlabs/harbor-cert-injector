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

package extractor

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	goharborv1beta1 "github.com/goharbor/harbor-operator/apis/goharbor.io/v1beta1"
	"github.com/szlabs/harbor-cert-injector/pkg/cert/extractor/cluster"
	"github.com/szlabs/harbor-cert-injector/pkg/cert/extractor/pi"
	"github.com/szlabs/harbor-cert-injector/pkg/cert/extractor/secret"
	"github.com/szlabs/harbor-cert-injector/pkg/types"
	mytypes "github.com/szlabs/harbor-cert-injector/pkg/types"
	packagev1alpha1 "github.com/vmware-tanzu/carvel-kapp-controller/pkg/apis/packaging/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Provider of extractor.
type Provider interface {
	// Extract the required info and put it into injection for injector using.
	// The controller.LastModification MUST be generated and bound.
	Extract(ctx context.Context, obj client.Object) (*types.Injection, error)
}

// ProviderFactory provides a extractor provider factory.
type ProviderFactory interface {
	// Get corresponding provider interface by the provided GVK of target resource.
	Get(GVK string) Provider
}

// Providers gives a provider factory for getting the related extractor provider.
func Providers(client client.Client) ProviderFactory {
	return &defaultFactory{
		client,
	}
}

type defaultFactory struct {
	client.Client
}

// Get implements ProviderFactory.
func (df *defaultFactory) Get(GVK string) Provider {
	if len(GVK) == 0 {
		return nil
	}

	switch GVK {
	case packagev1alpha1.SchemeGroupVersion.WithKind(mytypes.PackageInstall).String():
		return &pi.Provider{
			Client: df.Client,
		}
	case goharborv1beta1.GroupVersion.WithKind(mytypes.HarborCluster).String():
		return &cluster.Provider{
			Client: df.Client,
		}
	case corev1.SchemeGroupVersion.WithKind(mytypes.Secret).String():
		return &secret.Provider{
			Client: df.Client,
		}
	default:
		return nil
	}
}
