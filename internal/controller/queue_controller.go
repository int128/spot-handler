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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/int128/spot-handler/internal/spot"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
)

// QueueReconciler reconciles a Queue object
type QueueReconciler struct {
	ctrlclient.Client
	Scheme    *runtime.Scheme
	SQSClient SQSClient
}

// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=queues,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=queues/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=queues/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *QueueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var obj spothandlerv1.Queue
	if err := r.Get(ctx, req.NamespacedName, &obj); err != nil {
		return ctrl.Result{}, ctrlclient.IgnoreNotFound(err)
	}
	logger := ctrllog.FromContext(ctx, "sqsQueueURL", obj.Spec.URL)
	ctx = ctrllog.IntoContext(ctx, logger)

	receiveMessageOutput, err := r.SQSClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(obj.Spec.URL),
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     10,
	})
	if err != nil {
		logger.Error(err, "Failed to receive messages from the queue")
		return ctrl.Result{}, fmt.Errorf("failed to receive messages from the queue: %w", err)
	}
	messages := receiveMessageOutput.Messages
	if len(messages) == 0 {
		return ctrl.Result{RequeueAfter: 1 * time.Millisecond}, nil
	}
	logger.Info("Received messages from the queue", "count", len(messages))

	var wg sync.WaitGroup
	errs := make([]error, len(messages))
	for i, msg := range messages {
		wg.Add(1)
		go func(i int, msg sqstypes.Message) {
			defer wg.Done()
			if err := r.reconcileMessage(ctx, obj, msg); err != nil {
				errs[i] = err
			}
		}(i, msg)
	}
	return ctrl.Result{RequeueAfter: 1 * time.Millisecond}, errors.Join(errs...)
}

func (r *QueueReconciler) reconcileMessage(ctx context.Context, obj spothandlerv1.Queue, msg sqstypes.Message) error {
	logger := ctrllog.FromContext(ctx,
		"messageID", aws.ToString(msg.MessageId),
		"receiptHandle", aws.ToString(msg.ReceiptHandle),
	)
	ctx = ctrllog.IntoContext(ctx, logger)

	if err := r.createSpotInterruption(ctx, obj, msg); err != nil {
		return err
	}
	if _, err := r.SQSClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(obj.Spec.URL),
		ReceiptHandle: msg.ReceiptHandle,
	}); err != nil {
		logger.Error(err, "Failed to delete the message from queue")
		return fmt.Errorf("failed to delete the message from queue: %w", err)
	}
	logger.Info("Deleted the message from queue")
	return nil
}

func (r *QueueReconciler) createSpotInterruption(ctx context.Context, obj spothandlerv1.Queue, msg sqstypes.Message) error {
	logger := ctrllog.FromContext(ctx)

	spec, err := spot.Parse(aws.ToString(msg.Body))
	if err != nil {
		logger.Info("Ignored an invalid message", "error", err, "body", aws.ToString(msg.Body))
		return nil
	}
	spotInterruption := spothandlerv1.SpotInterruption{
		ObjectMeta: metav1.ObjectMeta{
			Name: spec.InstanceID,
		},
		Spec: *spec,
	}
	spec.Queue = spothandlerv1.QueueReference{Name: obj.Name}
	if err := ctrl.SetControllerReference(&obj, &spotInterruption, r.Scheme); err != nil {
		return fmt.Errorf("failed to set the controller reference from Queue to SpotInterruption: %w", err)
	}
	if err := r.Client.Create(ctx, &spotInterruption); err != nil {
		return ctrlclient.IgnoreAlreadyExists(fmt.Errorf("failed to create a SpotInterruption: %w", err))
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *QueueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&spothandlerv1.Queue{}).
		Complete(r)
}
