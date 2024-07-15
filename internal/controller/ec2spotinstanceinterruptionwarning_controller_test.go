/*
Copyright 2024.

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

package controller

import (
	"context"
	"time"

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
)

var _ = Describe("EC2SpotInstanceInterruptionWarning Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.TODO()

		It("should successfully reconcile the resource", func() {
			By("Creating a Node")
			Expect(k8sClient.Create(ctx, &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
				Spec: corev1.NodeSpec{
					ProviderID: "aws:///us-east-2a/i-1234567890abcdef0",
				},
			})).To(Succeed())

			By("Creating a Pod")
			Expect(k8sClient.Create(ctx, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					NodeName: "test-node",
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			})).To(Succeed())

			By("Creating an EC2SpotInstanceInterruptionWarning resource")
			Expect(k8sClient.Create(ctx, &spothandlerv1.EC2SpotInstanceInterruptionWarning{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ec2spotinstanceinterruptionwarning",
				},
				Spec: spothandlerv1.EC2SpotInstanceInterruptionWarningSpec{
					EventTime:        metav1.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
					InstanceID:       "i-1234567890abcdef0",
					AvailabilityZone: "us-east-2a",
				},
			})).To(Succeed())

			By("Reconciling the created resource")
			Eventually(func(g Gomega) {
				var ec2SpotInstanceInterruptionWarning spothandlerv1.EC2SpotInstanceInterruptionWarning
				g.Expect(k8sClient.Get(ctx, ktypes.NamespacedName{Name: "test-ec2spotinstanceinterruptionwarning"}, &ec2SpotInstanceInterruptionWarning)).To(Succeed())
				g.Expect(ec2SpotInstanceInterruptionWarning.Status.ProcessedTime.UTC()).To(Equal(time.Date(2022, 1, 1, 1, 1, 1, 0, time.UTC)))
			}).Should(Succeed())
		})
	})
})
