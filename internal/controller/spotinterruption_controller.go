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

const nodeProviderIDField = ".spec.providerID"

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

	if err := r.reconcile(ctx, obj); err != nil {
		return ctrl.Result{}, err
	}
	obj.Status.ReconciledAt = metav1.NewTime(r.Clock.Now())
	if err := r.Status().Update(ctx, &obj); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update the status of SpotInterruption: %w", err)
	}
	logger.Info("Successfully reconciled SpotInterruption")
	return ctrl.Result{}, nil
}

func (r *SpotInterruptionReconciler) reconcile(ctx context.Context, obj spothandlerv1.SpotInterruption) error {
	logger := ctrllog.FromContext(ctx)

	nodeProviderID := fmt.Sprintf("aws:///%s/%s", obj.Spec.AvailabilityZone, obj.Spec.InstanceID)
	var nodeList corev1.NodeList
	if err := r.List(ctx, &nodeList, ctrlclient.MatchingFields{nodeProviderIDField: nodeProviderID}); err != nil {
		return fmt.Errorf("failed to find Nodes: %w", err)
	}
	if len(nodeList.Items) == 0 {
		logger.Info("Node does not exist", "providerID", nodeProviderID)
		return nil
	}
	for _, node := range nodeList.Items {
		if err := r.createSpotInterruptedNode(ctx, obj, node); err != nil {
			return err
		}
	}
	return nil
}

func (r *SpotInterruptionReconciler) createSpotInterruptedNode(ctx context.Context, obj spothandlerv1.SpotInterruption, node corev1.Node) error {
	spotInterruptedNode := spothandlerv1.SpotInterruptedNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: node.Name,
		},
		Spec: spothandlerv1.SpotInterruptedNodeSpec{
			Node: corev1.LocalObjectReference{Name: node.Name},
		},
	}
	if err := ctrl.SetControllerReference(&obj, &spotInterruptedNode, r.Scheme); err != nil {
		return fmt.Errorf("failed to set the controller reference from SpotInterruption to SpotInterruptedNode: %w", err)
	}
	if err := r.Create(ctx, &spotInterruptedNode); err != nil {
		return ctrlclient.IgnoreAlreadyExists(fmt.Errorf("failed to create SpotInterruptedNode: %w", err))
	}
	return nil
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

	return ctrl.NewControllerManagedBy(mgr).
		For(&spothandlerv1.SpotInterruption{}).
		Complete(r)
}
