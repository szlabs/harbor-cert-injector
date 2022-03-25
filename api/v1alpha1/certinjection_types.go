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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CertInjectionSpec defines the desired state of CertInjection
type CertInjectionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	// ExternalDNS of the harbor registry.
	ExternalDNS string `json:"externalDNS"`

	// +kubebuilder:validation:Required
	// CertSecret is the name of the secret which contains the certificate.
	CertSecret corev1.LocalObjectReference `json:"certSecret"`
}

// CertInjectionStatus defines the observed state of CertInjection
type CertInjectionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Conditions of CertInjection.
	Conditions []CertInjectionCondition `json:"conditions,omitempty"`
	// CertSourceRef where the CA certification from.
	CertSourceRef *corev1.ObjectReference `json:"certSource,omitempty"`
	// Injector injects the CA cert into worker nodes where containerd is running.
	// Rely on a DaemonSet to do injection work.
	Injector *corev1.ObjectReference `json:"injector,omitempty"`
}

// CertInjectionCondition defines the observed condition of CertInjectionStatus.
type CertInjectionCondition struct {
	Type               string                 `json:"type"`
	Status             corev1.ConditionStatus `json:"status"`
	LastTransitionTime *metav1.Time           `json:"lastTransitionTime,omitempty"`
	Reason             string                 `json:"reason,omitempty"`
	Message            string                 `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CertInjection is the Schema for the certinjections API
type CertInjection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CertInjectionSpec   `json:"spec,omitempty"`
	Status CertInjectionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CertInjectionList contains a list of CertInjection
type CertInjectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CertInjection `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CertInjection{}, &CertInjectionList{})
}
