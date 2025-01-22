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

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
)

var _ = Describe("SpotInterruptedNode Controller", func() {
	Context("When reconciling a resource", func() {
		It("should successfully reconcile the resource", func() {
			ctx := context.TODO()

			By("Creating a Node resource")
			fixtureNode := corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-node-",
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

			By("Creating a SpotInterruptedNode resource")
			spotInterruptedNode := spothandlerv1.SpotInterruptedNode{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-spotinterruptednode-",
				},
				Spec: spothandlerv1.SpotInterruptedNodeSpec{
					Node:       corev1.LocalObjectReference{Name: fixtureNode.Name},
					Queue:      spothandlerv1.QueueReference{Name: "test-queue"},
					InstanceID: "i-1234567890abcdef0",
				},
			}
			Expect(k8sClient.Create(ctx, &spotInterruptedNode)).To(Succeed())

			By("Checking if SpotInterruptedNode is reconciled")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, ktypes.NamespacedName{Name: spotInterruptedNode.Name}, &spotInterruptedNode)).To(Succeed())
				g.Expect(spotInterruptedNode.Status.ReconciledAt.UTC()).To(Equal(fakeNow))
			})

			By("Checking if SpotInterruptedPod is created")
			var spotInterruptedPod spothandlerv1.SpotInterruptedPod
			Eventually(func() error {
				return k8sClient.Get(ctx,
					ktypes.NamespacedName{Name: fixturePod.Name, Namespace: fixturePod.Namespace}, &spotInterruptedPod)
			}).Should(Succeed())
			Expect(spotInterruptedPod.Spec.Node.Name).To(Equal(fixtureNode.Name))
			Expect(spotInterruptedPod.Spec.Queue.Name).To(Equal("test-queue"))
		})
	})
})
