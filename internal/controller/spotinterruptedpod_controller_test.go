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
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("SpotInterruptedPod Controller", func() {
	var fixturePod corev1.Pod

	BeforeEach(func() {
		ctx := context.TODO()

		By("Creating a Pod resource")
		fixturePod = corev1.Pod{
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
			Expect(ctrlclient.IgnoreNotFound(k8sClient.Delete(ctx, &fixturePod))).To(Succeed())
		})
	})

	Context("When reconciling a resource", func() {
		It("should successfully reconcile the resource", func() {
			ctx := context.TODO()

			By("Creating a Queue object")
			queue := spothandlerv1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-queue-",
				},
				Spec: spothandlerv1.QueueSpec{},
			}
			Expect(k8sClient.Create(ctx, &queue)).To(Succeed())
			DeferCleanup(func() {
				By("Deleting the Queue object")
				Expect(k8sClient.Delete(ctx, &queue)).To(Succeed())
			})

			By("Creating a SpotInterruption resource")
			spotInterruption := spothandlerv1.SpotInterruption{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-spotinterruption-",
				},
				Spec: spothandlerv1.SpotInterruptionSpec{
					InstanceID: "i-1234567890abcdef0",
					Queue:      spothandlerv1.QueueReferenceTo(queue),
				},
			}
			Expect(k8sClient.Create(ctx, &spotInterruption)).To(Succeed())
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, &spotInterruption)).To(Succeed())
			})

			By("Creating a SpotInterruptedPod resource")
			spotInterruptedPod := spothandlerv1.SpotInterruptedPod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-spotinterruptedpod-",
					Namespace:    "default",
				},
				Spec: spothandlerv1.SpotInterruptedPodSpec{
					Pod:              corev1.LocalObjectReference{Name: fixturePod.Name},
					Node:             corev1.LocalObjectReference{Name: "test-node"},
					SpotInterruption: spothandlerv1.SpotInterruptionReferenceTo(spotInterruption),
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

			By("Checking if the SpotInterruptedPodTermination is not created")
			var spotInterruptedPodTermination spothandlerv1.SpotInterruptedPodTermination
			err := k8sClient.Get(ctx, ktypes.NamespacedName{Name: spotInterruptedPod.Name, Namespace: spotInterruptedPod.Namespace}, &spotInterruptedPodTermination)
			Expect(err).To(HaveOccurred())
			Expect(kerrors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("When pod termination is enabled", func() {
		It("should terminate the Pod", func() {
			ctx := context.TODO()

			By("Creating a Queue object")
			queue := spothandlerv1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-queue-",
				},
				Spec: spothandlerv1.QueueSpec{
					SpotInterruption: spothandlerv1.QueueSpotInterruptionSpec{
						PodTermination: spothandlerv1.QueuePodTerminationSpec{
							Enabled:            true,
							DelaySeconds:       30,
							GracePeriodSeconds: ptr.To(int64(1)),
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &queue)).To(Succeed())
			DeferCleanup(func() {
				By("Deleting the Queue object")
				Expect(k8sClient.Delete(ctx, &queue)).To(Succeed())
			})

			By("Creating a SpotInterruption resource")
			spotInterruptionEventTimestamp := metav1.Date(2021, 7, 2, 3, 4, 5, 0, time.UTC)
			spotInterruption := spothandlerv1.SpotInterruption{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-spotinterruption-",
				},
				Spec: spothandlerv1.SpotInterruptionSpec{
					EventTimestamp: spotInterruptionEventTimestamp,
					InstanceID:     "i-1234567890abcdef0",
					Queue:          spothandlerv1.QueueReferenceTo(queue),
				},
			}
			Expect(k8sClient.Create(ctx, &spotInterruption)).To(Succeed())
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, &spotInterruption)).To(Succeed())
			})

			By("Creating a SpotInterruptedPod resource")
			spotInterruptedPod := spothandlerv1.SpotInterruptedPod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-spot-interrupted-pod-",
					Namespace:    "default",
				},
				Spec: spothandlerv1.SpotInterruptedPodSpec{
					Pod:              corev1.LocalObjectReference{Name: fixturePod.Name},
					Node:             corev1.LocalObjectReference{Name: "test-node"},
					SpotInterruption: spothandlerv1.SpotInterruptionReferenceTo(spotInterruption),
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

			By("Checking if the SpotInterruptedPodTermination is created")
			var spotInterruptedPodTermination spothandlerv1.SpotInterruptedPodTermination
			Expect(k8sClient.Get(ctx, ktypes.NamespacedName{Name: spotInterruptedPod.Name, Namespace: spotInterruptedPod.Namespace}, &spotInterruptedPodTermination)).To(Succeed())
			Expect(spotInterruptedPodTermination.Spec).To(Equal(spothandlerv1.SpotInterruptedPodTerminationSpec{
				TerminationTimestamp: metav1.NewTime(spotInterruptionEventTimestamp.Add(30 * time.Second).Local()),
				GracePeriodSeconds:   ptr.To(int64(1)),
				Pod:                  corev1.LocalObjectReference{Name: fixturePod.Name},
				Node:                 corev1.LocalObjectReference{Name: "test-node"},
				InstanceID:           "i-1234567890abcdef0",
			}))
		})
	})
})
