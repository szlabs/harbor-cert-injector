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
	appCatalogSecretRefKey  = "inline-values"
	tkgPackageSecretRefName = "harbor-default-values"
	appCatalogCaSecretName  = "harbor-ca-key-pair"
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

	// Find the name of the secret that contains the configuration values.
	secretRef := getValueSecret(pkgInstall)
	// Get the configuration values
	pvs, err := p.getValuesFromSecret(ctx, *secretRef)
	if err != nil {
		return nil, errs.Wrap("failed to get configuration values from the secret", err)
	}

	// There are three ways to set the CA cert, check it one by one.
	// Set in the `tlsCertificate` field.
	if pvs.TLSCertificate != nil && len(pvs.TLSCertificate.CACert) > 0 {
		return &mytypes.Injection{
			ExternalDNS: pvs.HostName,
			CACert:      []byte(pvs.TLSCertificate.CACert),
		}, nil
	}

	// Set with a separate secret,
	// or inject into a fixed secret by the cert-manager.
	caSecretRef := appCatalogCaSecretName
	if pvs.TLSCertificateSecretName != nil {
		caSecretRef = *pvs.TLSCertificateSecretName
	}

	CAContent, err := p.extractCAFromSecret(ctx, types.NamespacedName{
		Name:      caSecretRef,
		Namespace: pvs.Namespace,
	})

	if err != nil {
		return nil, errs.Wrap("failed to extract CA from the specified secret", err)
	}

	return &mytypes.Injection{
		ExternalDNS: pvs.HostName,
		CACert:      CAContent,
	}, nil
}

func (p *Provider) getValuesFromSecret(ctx context.Context, secret types.NamespacedName) (*packageValues, error) {
	vSecret := &corev1.Secret{}
	if err := p.Get(ctx, secret, vSecret); err != nil {
		return nil, errs.Wrap("failed to get the values secret object", err)
	}

	var decodedV []byte
	// There will be only 1 field.
	// Because the TKG package and TMC app catalog use different key and the key in TKG package is not fixed,
	// then we just get the first and only field.
	for _, encodedV := range vSecret.Data {
		if _, err := base64.StdEncoding.Decode(decodedV, encodedV); err != nil {
			return nil, errs.Wrap("failed to decode the base64 encoded configuration values", err)
		}

		break
	}

	// Unmarshal the required data.
	pvs := &packageValues{}
	if err := json.Unmarshal(decodedV, pvs); err != nil {
		return nil, errs.Wrap("failed to unmarshal configuration values", err)
	}

	return pvs, nil
}

func (p *Provider) extractCAFromSecret(ctx context.Context, secretRef types.NamespacedName) ([]byte, error) {
	caSecret := &corev1.Secret{}
	if err := p.Get(ctx, secretRef, caSecret); err != nil {
		return nil, errs.Wrap("failed to get the CA secret object", err)
	}

	var decodedV []byte
	if encodedV, ok := caSecret.Data[mytypes.CAKeyInSecret]; ok {
		if _, err := base64.StdEncoding.Decode(decodedV, encodedV); err != nil {
			return nil, errs.Wrap("failed to decode the base64 encoded CA content", err)
		}

		return decodedV, nil
	}

	return nil, errs.Errorf("missing %s in the secret data", mytypes.CAKeyInSecret)
}

func getValueSecret(pkgInstall *packagev1alpha1.PackageInstall) *types.NamespacedName {
	for _, v := range pkgInstall.Spec.Values {
		if v.SecretRef.Key == appCatalogSecretRefKey ||
			v.SecretRef.Name == tkgPackageSecretRefName {
			return &types.NamespacedName{
				Name:      v.SecretRef.Name,
				Namespace: pkgInstall.Namespace,
			}
		}
	}

	return nil
}
