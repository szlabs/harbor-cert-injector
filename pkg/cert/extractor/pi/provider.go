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

package pi

import (
	"context"
	"encoding/base64"
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/szlabs/harbor-cert-injector/pkg/errs"
	mytypes "github.com/szlabs/harbor-cert-injector/pkg/types"
	packagev1alpha1 "github.com/vmware-tanzu/carvel-kapp-controller/pkg/apis/packaging/v1alpha1"
)

const (
	appCatalogSecretRefKey = "inline-values"
	secretDataKey          = appCatalogSecretRefKey
	caDataKey              = "ca.crt"
	appCatalogCaSecretName = "harbor-ca-key-pair"
)

type packageValues struct {
	HostName                 string          `json:"hostname"`
	Namespace                string          `json:"namespace"`
	TLSCertificate           *tlsCertificate `json:"tlsCertificate,omitempty"`
	TLSCertificateSecretName *string         `json:"tlsCertificateSecretName,omitempty"`
}

type tlsCertificate struct {
	CACert string `json:"ca.crt"`
}

// Provider for extracting data from package installs.
type Provider struct {
	client.Client
}

// Extract implements extractor.Provider.
func (p *Provider) Extract(ctx context.Context, obj client.Object) (*mytypes.Injection, error) {
	pkgInstall, ok := obj.(*packagev1alpha1.PackageInstall)
	if !ok {
		return nil, errs.New("expect v1alph1.PackageInstall object")
	}

	// Find the configuration values.
	for _, v := range pkgInstall.Spec.Values {
		if v.SecretRef.Key == appCatalogSecretRefKey {
			p.getValuesFromSecret(ctx, types.NamespacedName{
				Name:      v.SecretRef.Name,
				Namespace: pkgInstall.Namespace,
			})
		}
	}

	return nil, nil
}

func (p *Provider) getValuesFromSecret(ctx context.Context, secret types.NamespacedName) error {
	vSecret := &corev1.Secret{}
	if err := p.Get(ctx, secret, vSecret); err != nil {
		return errs.Wrap("get values from secret error", err)
	}

	var decodedV []byte
	if encodedV, ok := vSecret.Data[secretDataKey]; ok {
		if _, err := base64.StdEncoding.Decode(decodedV, encodedV); err != nil {
			return errs.Wrap("decode secret values error", err)
		}
	}

	if len(decodedV) > 0 {
		pvs := &packageValues{}
		if err := json.Unmarshal(decodedV, pvs); err != nil {
			return errs.Wrap("unmarshal secret values error", err)
		}
	}

	return nil
}
