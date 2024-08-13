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
	"fmt"

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

const podNodeNameField = ".spec.nodeName"

// SpotInterruptedNodeReconciler reconciles a SpotInterruptedNode object
type SpotInterruptedNodeReconciler struct {
	ctrlclient.Client
	Scheme *runtime.Scheme
	Clock  clock.PassiveClock
}

// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=spotinterruptednodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=spotinterruptednodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=spotinterruptednodes/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SpotInterruptedNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrllog.FromContext(ctx)

	var obj spothandlerv1.SpotInterruptedNode
	if err := r.Get(ctx, req.NamespacedName, &obj); err != nil {
		return ctrl.Result{}, ctrlclient.IgnoreNotFound(err)
	}
	if !obj.Status.ReconciledAt.IsZero() {
		return ctrl.Result{}, nil
	}

	if err := r.reconcile(ctx, obj); err != nil {
		return ctrl.Result{}, err
	}
	obj.Status.ReconciledAt = metav1.NewTime(r.Clock.Now())
	if err := r.Status().Update(ctx, &obj); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update the status of SpotInterruptedNode: %w", err)
	}
	logger.Info("Successfully reconciled SpotInterruptedNode")
	return ctrl.Result{}, nil
}

func (r *SpotInterruptedNodeReconciler) reconcile(ctx context.Context, obj spothandlerv1.SpotInterruptedNode) error {
	var podList corev1.PodList
	if err := r.List(ctx, &podList, ctrlclient.MatchingFields{podNodeNameField: obj.Spec.Node.Name}); err != nil {
		return fmt.Errorf("failed to find Pods: %w", err)
	}
	for _, pod := range podList.Items {
		if err := r.createSpotInterruptedPod(ctx, obj, pod); err != nil {
			return err
		}
	}
	if err := r.createEvent(ctx, obj); err != nil {
		return err
	}
	return nil
}

func (r *SpotInterruptedNodeReconciler) createSpotInterruptedPod(ctx context.Context, obj spothandlerv1.SpotInterruptedNode, pod corev1.Pod) error {
	spotInterruptedPod := spothandlerv1.SpotInterruptedPod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		Spec: spothandlerv1.SpotInterruptedPodSpec{
			Pod:        corev1.LocalObjectReference{Name: pod.Name},
			Node:       corev1.LocalObjectReference{Name: obj.Spec.Node.Name},
			InstanceID: obj.Spec.InstanceID,
		},
	}
	if err := ctrl.SetControllerReference(&obj, &spotInterruptedPod, r.Scheme); err != nil {
		return fmt.Errorf("failed to set the controller reference from SpotInterruptedNode to SpotInterruptedPod: %w", err)
	}
	if err := r.Create(ctx, &spotInterruptedPod); err != nil {
		return ctrlclient.IgnoreAlreadyExists(fmt.Errorf("failed to create SpotInterruptedPod: %w", err))
	}
	return nil
}

func (r *SpotInterruptedNodeReconciler) createEvent(ctx context.Context, obj spothandlerv1.SpotInterruptedNode) error {
	logger := ctrllog.FromContext(ctx)

	var node corev1.Node
	if err := r.Get(ctx, ctrlclient.ObjectKey{Name: obj.Spec.Node.Name}, &node); err != nil {
		return ctrlclient.IgnoreNotFound(fmt.Errorf("failed to get the Node: %w", err))
	}
	ref, err := reference.GetReference(r.Scheme, &node)
	if err != nil {
		return fmt.Errorf("failed to get the reference of the Node: %w", err)
	}
	// We emit an event without the EventRecorder because:
	//  - Set the Host field.
	//  - Emit an event exactly once.
	source := corev1.EventSource{
		Component: "spotinterruptednode-controller",
		Host:      node.Name,
	}
	t := metav1.NewTime(r.Clock.Now())
	event := corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("spotinterruptednode-%s", obj.Name),
			Namespace: metav1.NamespaceDefault,
		},
		Source:              source,
		ReportingController: source.Component,
		ReportingInstance:   source.Host,
		InvolvedObject:      *ref,
		FirstTimestamp:      t,
		LastTimestamp:       t,
		Count:               1,
		Type:                corev1.EventTypeWarning,
		Reason:              "SpotInterrupted",
		Message:             fmt.Sprintf("Node %s of %s is interrupted", obj.Spec.Node.Name, obj.Spec.InstanceID),
	}
	if err := r.Create(ctx, &event); err != nil {
		return ctrlclient.IgnoreAlreadyExists(fmt.Errorf("failed to create an Event: %w", err))
	}
	logger.Info("Created an Event", "reason", event.Reason, "message", event.Message)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SpotInterruptedNodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, podNodeNameField,
		func(obj ctrlclient.Object) []string {
			pod := obj.(*corev1.Pod)
			if pod.Spec.NodeName == "" {
				return nil
			}
			return []string{pod.Spec.NodeName}
		},
	); err != nil {
		return fmt.Errorf("failed to create an index for field %s: %w", podNodeNameField, err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&spothandlerv1.SpotInterruptedNode{}).
		Complete(r)
}
