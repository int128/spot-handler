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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

var _ = Describe("SpotInterruption Controller", func() {
	Context("When reconciling a resource", func() {
		It("should successfully reconcile the resource", func() {
			ctx := context.TODO()

			By("Creating a Node resource")
			fixtureNode := corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-node-",
				},
				Spec: corev1.NodeSpec{
					ProviderID: "aws:///us-east-2a/i-00000000000000001",
				},
			}
			Expect(k8sClient.Create(ctx, &fixtureNode)).To(Succeed())

			By("Creating a Pod resource")
			fixturePod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-pod-",
					Namespace:    "default",
				},
				Spec: corev1.PodSpec{
					NodeName: fixtureNode.Name,
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &fixturePod)).To(Succeed())

			By("Creating a SpotInterruption resource")
			spotInterruption := spothandlerv1.SpotInterruption{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-spotinterruption-",
				},
				Spec: spothandlerv1.SpotInterruptionSpec{
					EventTimestamp:   metav1.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
					InstanceID:       "i-00000000000000001",
					AvailabilityZone: "us-east-2a",
					PodTermination: spothandlerv1.PodTerminationSpec{
						Enabled:            true,
						GracePeriodSeconds: ptr.To(int64(1)),
					},
				},
			}
			Expect(k8sClient.Create(ctx, &spotInterruption)).To(Succeed())

			By("Checking if SpotInterruption is reconciled")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, ktypes.NamespacedName{Name: spotInterruption.Name}, &spotInterruption)).To(Succeed())
				g.Expect(spotInterruption.Status.ReconciledAt.UTC()).To(Equal(fakeNow))
			}).Should(Succeed())

			By("Checking if SpotInterruptedNode is created")
			var spotInterruptedNode spothandlerv1.SpotInterruptedNode
			Eventually(func() error {
				return k8sClient.Get(ctx, ktypes.NamespacedName{Name: fixtureNode.Name}, &spotInterruptedNode)
			}).Should(Succeed())
			Expect(spotInterruptedNode.Spec.Node.Name).To(Equal(fixtureNode.Name))
			Expect(spotInterruptedNode.Spec.SpotInterruption.Name).To(Equal(spotInterruption.Name))
		})
	})

	Context("When the retention period is over", func() {
		It("should delete the resource", func() {
			ctx := context.TODO()

			By("Creating a SpotInterruption resource")
			spotInterruption := spothandlerv1.SpotInterruption{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-spotinterruption-",
				},
				Spec: spothandlerv1.SpotInterruptionSpec{
					EventTimestamp:   metav1.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
					InstanceID:       "i-00000000000000002",
					AvailabilityZone: "us-east-2a",
				},
			}
			Expect(k8sClient.Create(ctx, &spotInterruption)).To(Succeed())

			By("Waiting for the first reconciliation")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, ktypes.NamespacedName{Name: spotInterruption.Name}, &spotInterruption)).To(Succeed())
				g.Expect(spotInterruption.Status.ReconciledAt).ShouldNot(BeZero())
			}).Should(Succeed())

			By("Updating reconciledAt")
			spotInterruption.Status.ReconciledAt = metav1.Date(2021, 6, 30, 0, 0, 0, 0, time.UTC)
			Expect(k8sClient.Status().Update(ctx, &spotInterruption)).To(Succeed())

			By("Checking if the resource is deleted")
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, ktypes.NamespacedName{Name: spotInterruption.Name}, &spotInterruption)
				g.Expect(err).To(HaveOccurred())
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
			}).Should(Succeed())
		})
	})
})
