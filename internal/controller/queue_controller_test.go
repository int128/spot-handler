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
	_ "embed"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
)

//go:embed testdata/instance1.json
var messageInstance1 string

//go:embed testdata/instance2.json
var messageInstance2 string

var _ = Describe("Queue Controller", func() {
	var queue spothandlerv1.Queue

	BeforeEach(func() {
		ctx := context.TODO()
		By("Creating a Queue object")
		queue = spothandlerv1.Queue{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-queue-",
			},
			Spec: spothandlerv1.QueueSpec{
				URL: "https://sqs.us-east-2.amazonaws.com/123456789012/test-queue",
				SpotInterruption: spothandlerv1.QueueSpotInterruptionSpec{
					PodTermination: spothandlerv1.PodTerminationSpec{
						Enabled:            true,
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
	})

	Context("When a message is received", func() {
		It("should create a SpotInterruption resource", func() {
			ctx := context.TODO()

			By("Sending a message to the queue")
			mockSQSClient.reset(
				"https://sqs.us-east-2.amazonaws.com/123456789012/test-queue",
				[]sqstypes.Message{
					{
						Body:          aws.String(messageInstance1),
						ReceiptHandle: aws.String("ReceiptHandle-1"),
					},
					{
						Body:          aws.String(messageInstance2),
						ReceiptHandle: aws.String("ReceiptHandle-2"),
					},
				})
			Expect(mockSQSClient.messages).To(HaveLen(2))

			By("Checking if SpotInterruption#1 is created")
			var spotInterruption1 spothandlerv1.SpotInterruption
			Eventually(func() error {
				return k8sClient.Get(ctx, ktypes.NamespacedName{Name: "i-00000000000000001"}, &spotInterruption1)
			}).Should(Succeed())
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, &spotInterruption1)).To(Succeed())
			})

			Expect(spotInterruption1.Spec.EventTimestamp.UTC()).To(Equal(time.Date(2021, 2, 3, 14, 5, 6, 0, time.UTC)))
			Expect(spotInterruption1.Spec.InstanceID).To(Equal("i-00000000000000001"))
			Expect(spotInterruption1.Spec.AvailabilityZone).To(Equal("us-east-2a"))
			Expect(spotInterruption1.Spec.PodTermination).To(Equal(queue.Spec.SpotInterruption.PodTermination))

			By("Checking if SpotInterruption#2 is created")
			var spotInterruption2 spothandlerv1.SpotInterruption
			Eventually(func() error {
				return k8sClient.Get(ctx, ktypes.NamespacedName{Name: "i-00000000000000002"}, &spotInterruption2)
			}).Should(Succeed())
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, &spotInterruption2)).To(Succeed())
			})

			Expect(spotInterruption2.Spec.EventTimestamp.UTC()).To(Equal(time.Date(2021, 2, 3, 14, 5, 6, 0, time.UTC)))
			Expect(spotInterruption2.Spec.InstanceID).To(Equal("i-00000000000000002"))
			Expect(spotInterruption2.Spec.AvailabilityZone).To(Equal("us-east-2b"))
			Expect(spotInterruption2.Spec.PodTermination).To(Equal(queue.Spec.SpotInterruption.PodTermination))

			By("Checking if the queue becomes empty")
			Expect(mockSQSClient.messages).To(BeEmpty())
		})
	})

	Context("When a duplicated message is received", func() {
		It("should discard the message", func() {
			ctx := context.TODO()

			By("Sending a message to the queue")
			mockSQSClient.reset(
				"https://sqs.us-east-2.amazonaws.com/123456789012/test-queue",
				[]sqstypes.Message{{
					Body:          aws.String(messageInstance1),
					ReceiptHandle: aws.String("ReceiptHandle-1"),
				}},
			)

			By("Checking if SpotInterruption#1 is created")
			var spotInterruption1 spothandlerv1.SpotInterruption
			Eventually(func() error {
				return k8sClient.Get(ctx, ktypes.NamespacedName{Name: "i-00000000000000001"}, &spotInterruption1)
			}).Should(Succeed())
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, &spotInterruption1)).To(Succeed())
			})

			By("Checking if the queue is empty")
			Eventually(func() []sqstypes.Message { return mockSQSClient.messages }).Should(BeEmpty())

			By("Sending the same message to the queue")
			mockSQSClient.reset(
				"https://sqs.us-east-2.amazonaws.com/123456789012/test-queue",
				[]sqstypes.Message{{
					Body:          aws.String(messageInstance1),
					ReceiptHandle: aws.String("ReceiptHandle-1"),
				}},
			)

			By("Checking if the queue is still empty")
			Eventually(func() []sqstypes.Message { return mockSQSClient.messages }).Should(BeEmpty())
		})
	})

	Context("When an invalid message is received", func() {
		It("should discard the message", func() {
			By("Sending a message to the queue")
			mockSQSClient.reset(
				"https://sqs.us-east-2.amazonaws.com/123456789012/test-queue",
				[]sqstypes.Message{
					{
						Body:          aws.String("{}"), // invalid message
						ReceiptHandle: aws.String("ReceiptHandle-1"),
					},
				})
			Expect(mockSQSClient.messages).To(HaveLen(1))

			By("Checking if the queue becomes empty")
			Eventually(func() []sqstypes.Message { return mockSQSClient.messages }).Should(BeEmpty())
		})
	})
})
