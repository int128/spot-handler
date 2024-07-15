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
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
)

var _ = Describe("Queue Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.TODO()

		It("should successfully reconcile the resource", func() {
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
			Expect(mockSQSClient.messages).To(BeEmpty())
			mockSQSClient.append(sqstypes.Message{
				// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-instance-termination-notices.html
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
			})

			var ec2SpotInstanceInterruptionWarning spothandlerv1.EC2SpotInstanceInterruptionWarning
			Eventually(func() error {
				return k8sClient.Get(ctx,
					ktypes.NamespacedName{Name: "i-1234567890abcdef0"}, &ec2SpotInstanceInterruptionWarning)
			}).Should(Succeed())

			Expect(ec2SpotInstanceInterruptionWarning.Spec.EventTime.UTC()).To(Equal(
				time.Date(2021, 2, 3, 14, 5, 6, 0, time.UTC)))
			Expect(ec2SpotInstanceInterruptionWarning.Spec.InstanceID).To(Equal("i-1234567890abcdef0"))
			Expect(ec2SpotInstanceInterruptionWarning.Spec.AvailabilityZone).To(Equal("us-east-2a"))

			Expect(mockSQSClient.messages).To(BeEmpty())
		})
	})
})
