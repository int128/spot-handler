/*
Copyright 2025.

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
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
)

var _ = Describe("SpotInterruptedPodTermination Controller", func() {
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
			// Pod will be deleted by the controller in the test
			Expect(ctrlclient.IgnoreNotFound(k8sClient.Delete(ctx, &fixturePod))).To(Succeed())
		})
	})

	Context("When the termination timestamp is now", func() {
		It("should terminate the Pod immediately", func() {
			ctx := context.TODO()

			By("Creating a SpotInterruptedPodTermination resource")
			spotInterruptedPodTermination := spothandlerv1.SpotInterruptedPodTermination{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-spotinterruptedpodtermination-",
					Namespace:    "default",
				},
				Spec: spothandlerv1.SpotInterruptedPodTerminationSpec{
					TerminationTimestamp: metav1.NewTime(fakeNow),
					GracePeriodSeconds:   ptr.To(int64(30)),
					Pod:                  corev1.LocalObjectReference{Name: fixturePod.Name},
					Node:                 corev1.LocalObjectReference{Name: "test-node"},
					InstanceID:           "i-1234567890abcdef0",
				},
			}
			Expect(k8sClient.Create(ctx, &spotInterruptedPodTermination)).To(Succeed())
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, &spotInterruptedPodTermination)).To(Succeed())
			})

			By("Checking if the SpotInterruptedPodTermination is reconciled")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, ktypes.NamespacedName{Name: spotInterruptedPodTermination.Name, Namespace: spotInterruptedPodTermination.Namespace}, &spotInterruptedPodTermination)).To(Succeed())
				g.Expect(spotInterruptedPodTermination.Status.ReconciledAt).NotTo(BeZero())
			}).Should(Succeed())

			By("Checking if the Pod is terminated")
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, ktypes.NamespacedName{Name: fixturePod.Name, Namespace: fixturePod.Namespace}, &fixturePod)
				g.Expect(err).To(HaveOccurred())
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
			}).Should(Succeed())
		})
	})

	Context("When the termination timestamp is future", func() {
		It("should terminate the Pod later", func() {
			ctx := context.TODO()

			By("Creating a SpotInterruptedPodTermination resource")
			spotInterruptedPodTermination := spothandlerv1.SpotInterruptedPodTermination{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-spotinterruptedpodtermination-",
					Namespace:    "default",
				},
				Spec: spothandlerv1.SpotInterruptedPodTerminationSpec{
					TerminationTimestamp: metav1.NewTime(fakeNow.Add(1 * time.Second)),
					GracePeriodSeconds:   ptr.To(int64(30)),
					Pod:                  corev1.LocalObjectReference{Name: fixturePod.Name},
					Node:                 corev1.LocalObjectReference{Name: "test-node"},
					InstanceID:           "i-1234567890abcdef0",
				},
			}
			Expect(k8sClient.Create(ctx, &spotInterruptedPodTermination)).To(Succeed())
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, &spotInterruptedPodTermination)).To(Succeed())
			})

			By("Checking if the SpotInterruptedPodTermination is reconciled")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, ktypes.NamespacedName{Name: spotInterruptedPodTermination.Name, Namespace: spotInterruptedPodTermination.Namespace}, &spotInterruptedPodTermination)).To(Succeed())
				g.Expect(spotInterruptedPodTermination.Status.ReconciledAt).NotTo(BeZero())
			}).WithTimeout(3 * time.Second).Should(Succeed())
			Expect(spotInterruptedPodTermination.Status.ReconciledAt.UTC()).To(BeTemporally("=", fakeNow.Add(1*time.Second)))

			By("Checking if the Pod is terminated")
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, ktypes.NamespacedName{Name: fixturePod.Name, Namespace: fixturePod.Namespace}, &fixturePod)
				g.Expect(err).To(HaveOccurred())
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
			}).Should(Succeed())
		})
	})
})
