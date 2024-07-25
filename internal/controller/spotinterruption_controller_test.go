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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
)

var _ = Describe("SpotInterruption Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.TODO()

		It("should successfully reconcile the resource", func() {
			By("Creating a Node object")
			fixtureNode := corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-node-",
				},
				Spec: corev1.NodeSpec{
					ProviderID: "aws:///us-east-2a/i-00000000000000001",
				},
			}
			Expect(k8sClient.Create(ctx, &fixtureNode)).To(Succeed())

			By("Creating a Pod object")
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

			By("Creating an SpotInterruption object")
			spotInterruption := spothandlerv1.SpotInterruption{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-spotinterruption-",
				},
				Spec: spothandlerv1.SpotInterruptionSpec{
					EventAt:          metav1.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
					InstanceID:       "i-00000000000000001",
					AvailabilityZone: "us-east-2a",
				},
			}
			Expect(k8sClient.Create(ctx, &spotInterruption)).To(Succeed())

			By("Checking if the object is processed")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, ktypes.NamespacedName{Name: spotInterruption.Name}, &spotInterruption)).To(Succeed())
				g.Expect(spotInterruption.Status.ProcessedAt.UTC()).To(Equal(fakeNow))
			}).Should(Succeed())

			Expect(spotInterruption.Status.Interrupted.Nodes).To(Equal([]spothandlerv1.InterruptedNode{
				{
					Name: fixtureNode.Name,
				},
			}))
			Expect(spotInterruption.Status.Interrupted.Pods).To(Equal([]spothandlerv1.InterruptedPod{
				{
					Name:      fixturePod.Name,
					Namespace: fixturePod.Namespace,
				},
			}))
		})
	})
})
