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
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
)

const (
	nodeProviderIDField = ".spec.providerID"
	podNodeNameField    = ".spec.nodeName"
)

const spotInterruptionRetentionPeriod = 24 * time.Hour

// SpotInterruptionReconciler reconciles a SpotInterruption object
type SpotInterruptionReconciler struct {
	ctrlclient.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Clock    clock.PassiveClock
}

// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=spotinterruptions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=spotinterruptions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=spotinterruptions/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SpotInterruptionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrllog.FromContext(ctx)

	var obj spothandlerv1.SpotInterruption
	if err := r.Get(ctx, req.NamespacedName, &obj); err != nil {
		return ctrl.Result{}, ctrlclient.IgnoreNotFound(err)
	}

	if !obj.Status.ReconciledAt.IsZero() {
		expiry := obj.Status.ReconciledAt.Add(spotInterruptionRetentionPeriod)
		if r.Clock.Now().After(expiry) {
			if err := r.Client.Delete(ctx, &obj); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to delete an expired SpotInterruption: %w", err)
			}
			logger.Info("Deleted an expired SpotInterruption", "reconciledAt", obj.Status.ReconciledAt.Format(time.RFC3339))
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, nil
	}

	if result, err := r.reconcilePods(ctx, &obj); err != nil {
		return result, err
	}
	obj.Status.ReconciledAt = metav1.NewTime(r.Clock.Now())
	if err := r.Status().Update(ctx, &obj); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update the status of SpotInterruption: %w", err)
	}

	logger.Info("Successfully reconciled SpotInterruption")
	return ctrl.Result{}, nil
}

func (r *SpotInterruptionReconciler) reconcilePods(ctx context.Context, obj *spothandlerv1.SpotInterruption) (ctrl.Result, error) {
	logger := ctrllog.FromContext(ctx)

	nodeProviderID := fmt.Sprintf("aws:///%s/%s", obj.Spec.AvailabilityZone, obj.Spec.InstanceID)
	var nodeList corev1.NodeList
	if err := r.List(ctx, &nodeList, ctrlclient.MatchingFields{nodeProviderIDField: nodeProviderID}); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to find Nodes: %w", err)
	}
	if len(nodeList.Items) == 0 {
		logger.Info("Node does not exist", "providerID", nodeProviderID)
		return ctrl.Result{}, nil
	}

	for _, node := range nodeList.Items {
		r.Recorder.Eventf(&node, corev1.EventTypeWarning, "SpotInterrupted",
			"SpotInterrupted: Node %s, Instance %s in %s",
			node.Name, obj.Spec.InstanceID, obj.Spec.AvailabilityZone)

		var podList corev1.PodList
		if err := r.List(ctx, &podList, ctrlclient.MatchingFields{podNodeNameField: node.Name}); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to find Pods: %w", err)
		}
		for _, pod := range podList.Items {
			isDaemonSetPod := slices.ContainsFunc(pod.OwnerReferences,
				func(owner metav1.OwnerReference) bool {
					return owner.APIVersion == "apps/v1" && owner.Kind == "DaemonSet"
				})

			r.Recorder.Eventf(&pod, corev1.EventTypeWarning, "SpotInterrupted",
				"SpotInterrupted: Pod %s, Node %s, Instance %s in %s",
				pod.Name, node.Name, obj.Spec.InstanceID, obj.Spec.AvailabilityZone)

			//if err := r.Delete(ctx, &pod); err != nil {
			//	return err
			//}

			obj.Status.Interrupted.Pods = append(obj.Status.Interrupted.Pods, spothandlerv1.InterruptedPod{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				DaemonSet: isDaemonSetPod,
			})
		}

		obj.Status.Interrupted.Nodes = append(obj.Status.Interrupted.Nodes, spothandlerv1.InterruptedNode{
			Name: node.Name,
		})
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SpotInterruptionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Node{}, nodeProviderIDField,
		func(obj ctrlclient.Object) []string {
			node := obj.(*corev1.Node)
			if node.Spec.ProviderID == "" {
				return nil
			}
			return []string{node.Spec.ProviderID}
		},
	); err != nil {
		return fmt.Errorf("failed to create an index for field %s: %w", nodeProviderIDField, err)
	}

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
		For(&spothandlerv1.SpotInterruption{}).
		Complete(r)
}
