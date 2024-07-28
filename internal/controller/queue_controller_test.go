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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
)

var _ = Describe("Queue Controller", func() {
	Context("When a message is received", func() {
		It("should create a SpotInterruption resource", func() {
			ctx := context.TODO()
			const sqsQueueURL = "https://sqs.us-east-2.amazonaws.com/123456789012/test-queue"

			By("Creating a Queue object")
			Expect(k8sClient.Create(ctx, &spothandlerv1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-queue-",
				},
				Spec: spothandlerv1.QueueSpec{
					URL: sqsQueueURL,
				},
			})).To(Succeed())

			By("Sending a message to the queue")
			mockSQSClient.EXPECT().ReceiveMessage(
				mock.Anything,
				&sqs.ReceiveMessageInput{
					QueueUrl:            aws.String(sqsQueueURL),
					MaxNumberOfMessages: 10,
					WaitTimeSeconds:     10,
				},
			).Return(
				&sqs.ReceiveMessageOutput{
					Messages: []sqstypes.Message{
						// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-instance-termination-notices.html
						{
							Body: aws.String(`{
    "version": "0",
    "id": "12345678-1234-1234-1234-123456789012",
    "detail-type": "EC2 Spot Instance Interruption Warning",
    "source": "aws.ec2",
    "account": "123456789012",
    "time": "2021-02-03T14:05:06Z",
    "region": "us-east-2",
    "resources": ["arn:aws:ec2:us-east-2a:instance/i-1234567890abcdef0"],
    "detail": {
        "instance-id": "i-1234567890abcdef0",
        "instance-action": "action"
    }
}`),
							ReceiptHandle: aws.String("ReceiptHandle-1"),
						},
					},
				},
				nil,
			).Once()
			mockSQSClient.EXPECT().ReceiveMessage(
				mock.Anything,
				&sqs.ReceiveMessageInput{
					QueueUrl:            aws.String(sqsQueueURL),
					MaxNumberOfMessages: 10,
					WaitTimeSeconds:     10,
				},
			).Return(&sqs.ReceiveMessageOutput{Messages: nil}, nil)

			mockSQSClient.EXPECT().DeleteMessage(
				mock.Anything,
				&sqs.DeleteMessageInput{
					QueueUrl:      aws.String(sqsQueueURL),
					ReceiptHandle: aws.String("ReceiptHandle-1"),
				},
			).Return(
				&sqs.DeleteMessageOutput{},
				nil,
			).Once()

			By("Checking if a SpotInterruption resource is created")
			var spotInterruption spothandlerv1.SpotInterruption
			Eventually(func() error {
				return k8sClient.Get(ctx, ktypes.NamespacedName{Name: "i-1234567890abcdef0"}, &spotInterruption)
			}).Should(Succeed())

			Expect(spotInterruption.Spec.EventTimestamp.UTC()).To(Equal(time.Date(2021, 2, 3, 14, 5, 6, 0, time.UTC)))
			Expect(spotInterruption.Spec.InstanceID).To(Equal("i-1234567890abcdef0"))
			Expect(spotInterruption.Spec.AvailabilityZone).To(Equal("us-east-2a"))

			By("Checking if the message is deleted from the queue")
		})
	})
})
