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

package cluster

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	goharborv1beta1 "github.com/goharbor/harbor-operator/apis/goharbor.io/v1beta1"
	"github.com/szlabs/harbor-cert-injector/pkg/errs"
	mytypes "github.com/szlabs/harbor-cert-injector/pkg/types"
)

// Provider for extracting data from harborclusters.
type Provider struct {
	client.Client
}

func (p *Provider) Extract(ctx context.Context, obj client.Object) (*mytypes.Injection, error) {
	harbor, ok := obj.(*goharborv1beta1.HarborCluster)
	if !ok {
		return nil, errs.New("expected goharborv1beta1.HarborCluster obj but not")
	}

	if strings.TrimSpace(harbor.Spec.ExternalURL) == "" {
		return nil, errs.Errorf("external URL is not configured for harbor cluster %s:%s", harbor.Namespace, harbor.Name)
	}

	if harbor.Spec.Expose.Core.TLS == nil {
		return nil, errs.Wrap(fmt.Sprintf("harbor cluster %s:%s", harbor.Namespace, harbor.Name), errs.TLSNotEnabledError)
	}

	certRef := harbor.Spec.Expose.Core.TLS.CertificateRef

	caCert := &corev1.Secret{}
	if err := p.Get(ctx, types.NamespacedName{
		Name:      certRef,
		Namespace: harbor.Namespace,
	}, caCert); err != nil {
		return nil, errs.Wrap("get CA secret of harbor cluster error", err)
	}

	return &mytypes.Injection{
		ExternalDNS: strings.TrimPrefix(harbor.Spec.ExternalURL, "https://"),
		CACert:      caCert.Data["ca.crt"],
	}, nil
}
