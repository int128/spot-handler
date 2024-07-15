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
	"github.com/int128/spot-handler/internal/spot"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
)

const (
	nodeProviderIDField = ".spec.providerID"
	podNodeNameField    = ".spec.nodeName"
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

// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

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
		go func() {
			defer wg.Done()
			notice, err := spot.Parse(aws.ToString(message.Body))
			if err != nil {
				logger.Error(err, "Dropped the invalid message", "body", aws.ToString(message.Body))
				return
			}
			if err := r.processNotice(ctx, notice); err != nil {
				errs[i] = err
				return
			}
			if _, err := r.SQSClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(queueObj.Spec.URL),
				ReceiptHandle: message.ReceiptHandle,
			}); err != nil {
				errs[i] = err
				return
			}
		}()
	}
	return ctrl.Result{RequeueAfter: 1 * time.Millisecond}, errors.Join(errs...)
}

func (r *QueueReconciler) processNotice(ctx context.Context, notice spot.Notice) error {
	nodeProviderID := fmt.Sprintf("aws:///%s/%s", notice.AvailabilityZone, notice.Detail.InstanceID)

	var nodeList corev1.NodeList
	if err := r.List(ctx, &nodeList, client.MatchingFields{nodeProviderIDField: nodeProviderID}); err != nil {
		return fmt.Errorf("could not list nodes: %w", err)
	}
	if len(nodeList.Items) == 0 {
		return nil
	}
	node := nodeList.Items[0]
	r.Recorder.AnnotatedEventf(&node,
		map[string]string{
			"host": notice.Detail.InstanceID,
		},
		corev1.EventTypeWarning, "SpotInterrupted",
		"Instance %s is spot interrupted", notice.Detail.InstanceID)

	var podList corev1.PodList
	if err := r.List(ctx, &podList, client.MatchingFields{podNodeNameField: node.Name}); err != nil {
		return fmt.Errorf("could not list pods: %w", err)
	}
	if len(podList.Items) == 0 {
		return nil
	}
	for _, pod := range podList.Items {
		r.Recorder.AnnotatedEventf(&pod,
			map[string]string{
				"host": notice.Detail.InstanceID,
			},
			corev1.EventTypeWarning, "SpotInterrupted",
			"Instance %s is spot interrupted", notice.Detail.InstanceID)
		//if err := r.Delete(ctx, &pod); err != nil {
		//	return err
		//}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *QueueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Node{}, nodeProviderIDField,
		func(obj client.Object) []string {
			node := obj.(*corev1.Node)
			if node.Spec.ProviderID == "" {
				return nil
			}
			return []string{node.Spec.ProviderID}
		},
	); err != nil {
		return fmt.Errorf("could not create an index for field %s: %w", nodeProviderIDField, err)
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, podNodeNameField,
		func(obj client.Object) []string {
			pod := obj.(*corev1.Pod)
			if pod.Spec.NodeName == "" {
				return nil
			}
			return []string{pod.Spec.NodeName}
		},
	); err != nil {
		return fmt.Errorf("could not create an index for field %s: %w", podNodeNameField, err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&spothandlerv1.Queue{}).
		Complete(r)
}
