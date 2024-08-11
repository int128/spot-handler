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

var _ = Describe("SpotInterruptedPod Controller", func() {
	Context("When reconciling a resource", func() {
		It("should successfully reconcile the resource", func() {
			ctx := context.TODO()

			By("Creating a SpotInterruptedPod resource")
			spotInterruptedPod := spothandlerv1.SpotInterruptedPod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-pod-",
					Namespace:    "default",
				},
				Spec: spothandlerv1.SpotInterruptedPodSpec{
					Pod:        corev1.LocalObjectReference{Name: "test-pod"},
					Node:       corev1.LocalObjectReference{Name: "test-node"},
					InstanceID: "i-1234567890abcdef0",
				},
			}
			Expect(k8sClient.Create(ctx, &spotInterruptedPod)).To(Succeed())

			By("Checking if SpotInterruptedPod is reconciled")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, ktypes.NamespacedName{Name: spotInterruptedPod.Name}, &spotInterruptedPod)).To(Succeed())
				g.Expect(spotInterruptedPod.Status.ReconciledAt.UTC()).To(Equal(fakeNow))
			})
		})
	})
})
