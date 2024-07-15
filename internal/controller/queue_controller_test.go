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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
)

var _ = Describe("Queue Controller", func() {
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

			By("Creating a Queue resource")
			Expect(k8sClient.Create(ctx, &spothandlerv1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-queue",
				},
				Spec: spothandlerv1.QueueSpec{
					URL: "https://sqs.us-east-2.amazonaws.com/123456789012/test-queue",
				},
			})).To(Succeed())

			By("Reconciling the created resource")
			//time.Sleep(1 * time.Second)
			//Expect(false).To(BeTrue())
		})
	})
})
