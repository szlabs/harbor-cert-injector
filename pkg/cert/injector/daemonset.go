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
	"fmt"

	"github.com/szlabs/harbor-cert-injector/api/v1alpha1"
	"github.com/szlabs/harbor-cert-injector/pkg/errs"
	mytypes "github.com/szlabs/harbor-cert-injector/pkg/types"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	injectorImage       = "ghcr.io/szlabs/cert-injector:v0.1"
	containerdCertsPath = "/etc/containerd/certs.d"
	containerdTomlPath  = "/etc/containerd/config.toml"
	etcContainerd       = "/etc/containerd"
	dsNamePrefix        = "cert-injection-ds"
)

var terminationGracePeriodSeconds int64 = 30

// provider for doing injection through daemon set.
type provider struct {
	client.Client
	scheme *runtime.Scheme
}

// NewDaemonSetProvider news a daemonset provider.
func NewDaemonSetProvider(client client.Client, scheme *runtime.Scheme) Provider {
	return &provider{
		Client: client,
		scheme: scheme,
	}
}

// Inject implements injector.Provider.
func (p *provider) Inject(ctx context.Context, injection *v1alpha1.CertInjection) error {
	if injection == nil {
		return errs.New("nil cert injection obj")
	}

	dsCR := p.DesiredInjector(injection)

	// Set owner reference.
	if err := controllerutil.SetOwnerReference(injection, dsCR, p.scheme); err != nil {
		return errs.Wrap("set owner reference of ds error", err)
	}

	// Create the ds now.
	if err := p.Create(ctx, dsCR); err != nil {
		return errs.Wrap("create ds error", err)
	}

	// Get the created ds again.
	ds := &appv1.DaemonSet{}
	if err := p.Get(ctx, types.NamespacedName{
		Namespace: dsCR.Namespace,
		Name:      dsCR.Name,
	}, ds); err != nil {
		return errs.Wrap("get ds error", err)
	}

	// Get the object reference.
	objRef, err := reference.GetReference(p.scheme, ds)
	if err != nil {
		return errs.Wrap("get ds reference error", err)
	}

	injection.Status.Injector = objRef
	conditions := []v1alpha1.CertInjectionCondition{
		{
			Type:    mytypes.ConditionInjector,
			Status:  corev1.ConditionTrue,
			Message: "Injector has been created",
		}, {
			Type:   mytypes.ConditionReady,
			Status: corev1.ConditionTrue,
		},
	}
	injection.Status.Conditions = append(injection.Status.Conditions, conditions...)
	if err := p.Status().Update(ctx, injection); err != nil {
		return errs.Wrap("update status of cert injection error", err)
	}

	return nil
}

// DesiredInjector implements injector.Provider.
func (p *provider) DesiredInjector(injection *v1alpha1.CertInjection) *appv1.DaemonSet {
	if injection == nil {
		return nil
	}

	return &appv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName(injection.Name),
			Namespace: injection.Namespace,
			Labels: map[string]string{
				"k8s-app": "cert-auto-injector",
			},
			Annotations: map[string]string{
				mytypes.InjectionVersionAnnotationKey: injection.GetResourceVersion(),
			},
		},
		Spec: appv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": dsName(injection.Name),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name": dsName(injection.Name),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "cert-injector",
							Image: injectorImage,
							Command: []string{
								"inject",
							},
							Args: []string{
								"-r",
								injection.Spec.ExternalDNS,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "etc-containerd",
									MountPath: etcContainerd,
								},
								{
									Name:      "ca-cert",
									MountPath: "/tmp",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "etc-containerd",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: etcContainerd,
								},
							},
						},
						{
							Name: "ca-cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: injection.Spec.CertSecret.Name,
								},
							},
						},
					},
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
				},
			},
		},
	}
}

func dsName(name string) string {
	return fmt.Sprintf("%s-%s", dsNamePrefix, name)
}
