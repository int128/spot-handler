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
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
)

// SQSReconciler reconciles a SQS object
type SQSReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
	SQSClient *sqs.Client
}

// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=sqs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=sqs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=sqs/finalizers,verbs=update
// +kubebuilder:rbac:groups=,resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SQSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var sqsObj spothandlerv1.SQS
	if err := r.Get(ctx, req.NamespacedName, &sqsObj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	receiveMessageOutput, err := r.SQSClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(sqsObj.Spec.QueueURL),
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     10,
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	if len(receiveMessageOutput.Messages) == 0 {
		return ctrl.Result{RequeueAfter: 1 * time.Millisecond}, nil
	}
	logger.Info("Received message", "count", len(receiveMessageOutput.Messages))

	var errs []error
	for _, message := range receiveMessageOutput.Messages {
		if err := r.processMessage(ctx, message); err != nil {
			errs = append(errs, err)
		}
		if _, err := r.SQSClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
			QueueUrl:      aws.String(sqsObj.Spec.QueueURL),
			ReceiptHandle: message.ReceiptHandle,
		}); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: 1 * time.Millisecond}, nil
}

type EC2SpotInstanceInterruptionNotice struct {
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-instance-termination-notices.html
	Region string                                  `json:"region,omitempty"`
	Detail EC2SpotInstanceInterruptionNoticeDetail `json:"detail,omitempty"`
}

type EC2SpotInstanceInterruptionNoticeDetail struct {
	InstanceID     string `json:"instance-id,omitempty"`
	InstanceAction string `json:"instance-action,omitempty"`
}

func (r *SQSReconciler) processMessage(ctx context.Context, message sqstypes.Message) error {
	body := aws.ToString(message.Body)
	var notice = EC2SpotInstanceInterruptionNotice{}
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&notice); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	if notice.Detail.InstanceAction != "terminate" {
		return nil
	}
	return r.processNotice(ctx, notice)
}

func (r *SQSReconciler) processNotice(ctx context.Context, notice EC2SpotInstanceInterruptionNotice) error {
	// https://repost.aws/knowledge-center/eks-terminated-node-instance-id
	nodeProviderID := fmt.Sprintf("aws:///%s/%s", notice.Region, notice.Detail.InstanceID)

	var nodeList corev1.NodeList
	if err := r.List(ctx, &nodeList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("spec.providerID", nodeProviderID),
	}); err != nil {
		return fmt.Errorf("could not list nodes: %w", err)
	}
	if len(nodeList.Items) == 0 {
		return nil
	}
	node := nodeList.Items[0]

	var podList corev1.PodList
	if err := r.List(ctx, &podList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("spec.nodeName", node.Name),
	}); err != nil {
		return fmt.Errorf("could not list pods: %w", err)
	}
	if len(podList.Items) == 0 {
		return nil
	}

	for _, pod := range podList.Items {
		r.Recorder.Eventf(&pod, corev1.EventTypeWarning, "SpotInterrupted",
			"Pod %s is interrupted", pod.Name)
		//if err := r.Delete(ctx, &pod); err != nil {
		//	return err
		//}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SQSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&spothandlerv1.SQS{}).
		Complete(r)
}
