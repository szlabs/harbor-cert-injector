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

package types

const (
	// CAKeyInSecret ...
	CAKeyInSecret = "ca.crt"

	// OwnerAnnotationKey ...
	OwnerAnnotationKey = "registry.goharbor.io/uri"
	// InjectionVersionAnnotationKey ...
	InjectionVersionAnnotationKey = "injection.goharbor.io/version"
	// LastUpdateTimestampAnnotationKey ...
	LastUpdateTimestampAnnotationKey = "goharbor.io/last-updated"

	// HarborCluster kind.
	HarborCluster = "HarborCluster"
	// PackageInstall kind.
	PackageInstall = "PackageInstall"
	// Secret kind.
	Secret = "Secret"
	// CertInjection kind.
	CertInjection = "CertInjection"
	// ConditionReady ...
	ConditionReady = "Ready"
	// ConditionInjector ...
	ConditionInjector = "Injector Ready"
	// ConditionCAReady ...
	ConditionCAReady = "CA Secret Ready"
)

// Injection includes the related info extracted from the certificate source and
// used by the injector to do the cert injection.
type Injection struct {
	// ExternalDNS of the harbor registry.
	ExternalDNS string
	// CACert is certificate content.
	CACert []byte
}
