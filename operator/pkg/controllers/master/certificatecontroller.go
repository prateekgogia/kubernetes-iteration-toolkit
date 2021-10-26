/*
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

package master

import (
	"context"
	"fmt"

	"github.com/awslabs/kit/operator/pkg/apis/controlplane/v1alpha1"
	"github.com/awslabs/kit/operator/pkg/utils/imageprovider"
	"knative.dev/pkg/ptr"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) reconcileCertsControllerDaemonSet(ctx context.Context, controlPlane *v1alpha1.ControlPlane) error {
	vpcID, err := c.cloudProvider.VPCID()
	if err != nil {
		return fmt.Errorf("getting vpc ID, %w", err)
	}
	accountID, err := c.cloudProvider.ID()
	if err != nil {
		return fmt.Errorf("getting account ID, %w", err)
	}
	return c.kubeClient.EnsurePatch(ctx, &appsv1.DaemonSet{},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "eks-certificates-controller",
				Namespace: controlPlane.Namespace,
				Labels:    certControllerLabels(),
			},
			Spec: appsv1.DaemonSetSpec{
				UpdateStrategy: appsv1.DaemonSetUpdateStrategy{Type: appsv1.RollingUpdateDaemonSetStrategyType},
				Selector: &metav1.LabelSelector{
					MatchLabels: certControllerLabels(),
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: certControllerLabels(),
					},
					Spec: v1.PodSpec{
						HostNetwork:  true,
						NodeSelector: APIServerLabels(controlPlane.ClusterName()),
						Tolerations:  []v1.Toleration{{Operator: v1.TolerationOpExists}},
						Containers: []v1.Container{{
							Name:  "eks-certificates-controller",
							Image: imageprovider.CertificatesController(),
							Env: []v1.EnvVar{{
								Name:  "AWS_EXECUTION_ENV",
								Value: "eks-certificates-controller",
							}},
							Command: []string{
								"go-runner",
								"--also-stdout=true",
								"--redirect-stderr=true",
								"/eks-certificates-controller",
								"--enable-signer=true",
								"--kubeconfig=/etc/kubernetes/kubeconfig/eks-certificates-controller.kubeconfig",
								"--signing-cert-file=/etc/kubernetes/pki/ca/ca.crt",
								"--signing-key-file=/etc/kubernetes/pki/ca/ca.key",
								"--signing-duration=8760h",
								"--role-arn=arn:aws:iam::" + accountID + ":role/CertificateControllerRole",
								"--vpc-id=" + vpcID,
								"--per-node-rate-limit-duration=10m",
								"--threadiness=10",
								"--logtostderr",
							},
							// SecurityContext: &v1.SecurityContext{AllowPrivilegeEscalation: ptr.Bool(true)},
							VolumeMounts: []v1.VolumeMount{{
								Name:      "config",
								MountPath: "/etc/kubernetes/kubeconfig",
							}, {
								Name:      "client-ca-file",
								MountPath: "/etc/kubernetes/pki/ca",
							}},
						}},
						Volumes: []v1.Volume{{
							Name: "config",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									DefaultMode: ptr.Int32(0444),
									SecretName:  CertificatesControllerSecretNameFor(controlPlane.ClusterName()),
									Items: []v1.KeyToPath{{
										Key:  "config",
										Path: "eks-certificates-controller.kubeconfig",
									}},
								},
							},
						}, {
							Name: "client-ca-file",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName:  RootCASecretNameFor(controlPlane.ClusterName()),
									DefaultMode: ptr.Int32(0444),
									Items: []v1.KeyToPath{{
										Key:  "public",
										Path: "ca.crt",
									}, {
										Key:  "private",
										Path: "ca.key",
									}},
								},
							},
						}},
					},
				},
			},
		},
	)
}

func certControllerLabels() map[string]string {
	return map[string]string{
		"component": "eks-certificates-controller",
	}
}
