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
	"github.com/szlabs/harbor-cert-injector/pkg/errs"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	injectorImage       = "ghcr.io/szlabs/cert-injector:v0.1"
	containerdCertsPath = "/etc/containerd/certs.d"
	containerdTomlPath  = "/etc/containerd/config.toml"
	etcContainerd       = "/etc/containerd"
)

var terminationGracePeriodSeconds int64 = 30

// Injector for injecting self-signed CA via Daemonset.
type Injector interface {
	// For which registry.
	// The DNS/hostname of the registry.
	For(registry string) Injector

	// LiveInNS identify which NS the underlying daemon set will live in.
	LiveInNS(ns string) Injector

	// Inject the specified CA certificate.
	// Cert is the data bytes of the self-signed certificate.
	Inject(cert string) error
}

type defaultInjector struct {
	registry string
	ns       string
}

// New a default injector.
func New() Injector {
	return &defaultInjector{}
}

// For implements Injector.For
func (di *defaultInjector) For(registry string) Injector {
	di.registry = registry
	return di
}

// Inject implements Injector.Inject
func (di *defaultInjector) Inject(cert string) error {
	if len(cert) == 0 {
		return errs.New("empty cert")
	}

	return nil
}

// LiveInNS implements Injector.LiveInNS
func (di *defaultInjector) LiveInNS(ns string) Injector {
	di.ns = ns
	return di
}

func (di *defaultInjector) createDaemonSet() *appv1.DaemonSet {
	return &appv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "",
			Namespace: di.ns,
			Labels: map[string]string{
				"k8s-app": "cert-auto-injection",
			},
		},
		Spec: appv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": "",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name": "",
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
								di.registry,
								"-ca",
								"ca_secret",
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
									SecretName: "secret-name",
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
