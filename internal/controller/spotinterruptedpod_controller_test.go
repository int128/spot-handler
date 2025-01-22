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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("SpotInterruptedPod Controller", func() {
	Context("When reconciling a resource", func() {
		It("should successfully reconcile the resource", func() {
			ctx := context.TODO()

			By("Creating a Pod resource")
			fixturePod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-pod-",
					Namespace:    "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &fixturePod)).To(Succeed())
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, &fixturePod)).To(Succeed())
			})

			By("Creating a SpotInterruptedPod resource")
			spotInterruptedPod := spothandlerv1.SpotInterruptedPod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-spot-interrupted-pod-",
					Namespace:    "default",
				},
				Spec: spothandlerv1.SpotInterruptedPodSpec{
					Pod:        corev1.LocalObjectReference{Name: fixturePod.Name},
					Node:       corev1.LocalObjectReference{Name: "test-node"},
					InstanceID: "i-1234567890abcdef0",
				},
			}
			Expect(k8sClient.Create(ctx, &spotInterruptedPod)).To(Succeed())
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, &spotInterruptedPod)).To(Succeed())
			})

			By("Checking if the SpotInterruptedPod is reconciled")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, ktypes.NamespacedName{Name: spotInterruptedPod.Name, Namespace: spotInterruptedPod.Namespace}, &spotInterruptedPod)).To(Succeed())
				g.Expect(spotInterruptedPod.Status.ReconciledAt.UTC()).To(Equal(fakeNow))
			}).Should(Succeed())

			By("Checking if the Pod is not terminated")
			Expect(k8sClient.Get(ctx, ktypes.NamespacedName{Name: fixturePod.Name, Namespace: fixturePod.Namespace}, &fixturePod)).To(Succeed())
		})
	})

	Context("When pod termination is enabled", func() {
		It("should terminate the Pod", func() {
			ctx := context.TODO()

			By("Creating a Queue resource")
			queue := spothandlerv1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-queue-",
				},
				Spec: spothandlerv1.QueueSpec{
					PodTermination: spothandlerv1.QueuePodTerminationSpec{
						Enabled: true,
					},
				},
			}
			Expect(k8sClient.Create(ctx, &queue)).To(Succeed())
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, &queue)).To(Succeed())
			})

			By("Creating a Pod resource")
			fixturePod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-pod-",
					Namespace:    "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &fixturePod)).To(Succeed())
			DeferCleanup(func() {
				// Pod will be deleted by the controller in the test
				Expect(ctrlclient.IgnoreNotFound(k8sClient.Delete(ctx, &fixturePod))).To(Succeed())
			})

			By("Creating a SpotInterruptedPod resource")
			spotInterruptedPod := spothandlerv1.SpotInterruptedPod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-spot-interrupted-pod-",
					Namespace:    "default",
				},
				Spec: spothandlerv1.SpotInterruptedPodSpec{
					Pod:        corev1.LocalObjectReference{Name: fixturePod.Name},
					Node:       corev1.LocalObjectReference{Name: "test-node"},
					Queue:      spothandlerv1.QueueReference{Name: queue.Name},
					InstanceID: "i-1234567890abcdef0",
				},
			}
			Expect(k8sClient.Create(ctx, &spotInterruptedPod)).To(Succeed())
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, &spotInterruptedPod)).To(Succeed())
			})

			By("Checking if the SpotInterruptedPod is reconciled")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, ktypes.NamespacedName{Name: spotInterruptedPod.Name, Namespace: spotInterruptedPod.Namespace}, &spotInterruptedPod)).To(Succeed())
				g.Expect(spotInterruptedPod.Status.ReconciledAt).NotTo(BeZero())
			}).Should(Succeed())

			By("Checking if the Pod is terminated")
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, ktypes.NamespacedName{Name: fixturePod.Name, Namespace: fixturePod.Namespace}, &fixturePod)
				g.Expect(err).To(HaveOccurred())
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
			}).Should(Succeed())
		})
	})
})
