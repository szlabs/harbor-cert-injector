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

package injector

import (
	"context"

	appv1 "k8s.io/api/apps/v1"

	"github.com/szlabs/harbor-cert-injector/api/v1alpha1"
)

// Provider for injecting self-signed CA.
type Provider interface {
	// Inject the specified CA certificate.
	// Cert is the data bytes of the self-signed certificate.
	// The underlying injector will be created and updated into the injection status object.
	Inject(ctx context.Context, injection *v1alpha1.CertInjection) error

	// DesiredInjector indicates the desired injector object align with the provided injection.
	DesiredInjector(injection *v1alpha1.CertInjection) *appv1.DaemonSet
}
