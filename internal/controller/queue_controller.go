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
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/int128/spot-handler/internal/spot"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
)

// QueueReconciler reconciles a Queue object
type QueueReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
	SQSClient SQSClient
}

// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=queues,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=queues/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=queues/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *QueueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var queueObj spothandlerv1.Queue
	if err := r.Get(ctx, req.NamespacedName, &queueObj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	receiveMessageOutput, err := r.SQSClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueObj.Spec.URL),
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     10,
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	messages := receiveMessageOutput.Messages
	if len(messages) == 0 {
		return ctrl.Result{RequeueAfter: 1 * time.Millisecond}, nil
	}
	logger.Info("Received message", "count", len(messages))

	var wg sync.WaitGroup
	errs := make([]error, len(messages))
	for i, message := range messages {
		wg.Add(1)
		go func(i int, message sqstypes.Message) {
			defer wg.Done()
			if err := r.reconcileMessage(ctx, queueObj, message); err != nil {
				errs[i] = err
			}
		}(i, message)
	}
	return ctrl.Result{RequeueAfter: 1 * time.Millisecond}, errors.Join(errs...)
}

func (r *QueueReconciler) reconcileMessage(ctx context.Context, queueObj spothandlerv1.Queue, message sqstypes.Message) error {
	logger := log.FromContext(ctx)

	spec, err := spot.Parse(aws.ToString(message.Body))
	if err != nil {
		logger.Info("Dropped an invalid message", "error", err, "body", aws.ToString(message.Body))
		return nil
	}
	spotInterruption := spothandlerv1.SpotInterruption{
		ObjectMeta: metav1.ObjectMeta{
			Name: spec.InstanceID,
		},
		Spec: *spec,
	}
	if err := ctrl.SetControllerReference(&queueObj, &spotInterruption, r.Scheme); err != nil {
		return err
	}
	if err := r.Client.Create(ctx, &spotInterruption); err != nil {
		return err
	}
	if _, err := r.SQSClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueObj.Spec.URL),
		ReceiptHandle: message.ReceiptHandle,
	}); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *QueueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&spothandlerv1.Queue{}).
		Complete(r)
}
