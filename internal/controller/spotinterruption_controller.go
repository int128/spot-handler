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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
)

const (
	nodeProviderIDField = ".spec.providerID"
	podNodeNameField    = ".spec.nodeName"
)

// SpotInterruptionReconciler reconciles a SpotInterruption object
type SpotInterruptionReconciler struct {
	client.Client
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
	var obj spothandlerv1.SpotInterruption
	if err := r.Get(ctx, req.NamespacedName, &obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if result, err := r.process(ctx, obj); err != nil {
		return result, err
	}

	obj.Status.ProcessedAt = metav1.NewTime(r.Clock.Now())
	if err := r.Status().Update(ctx, &obj); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *SpotInterruptionReconciler) process(ctx context.Context, obj spothandlerv1.SpotInterruption) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	nodeProviderID := fmt.Sprintf("aws:///%s/%s", obj.Spec.AvailabilityZone, obj.Spec.InstanceID)
	var nodeList corev1.NodeList
	if err := r.List(ctx, &nodeList, client.MatchingFields{nodeProviderIDField: nodeProviderID}); err != nil {
		return ctrl.Result{}, err
	}
	if len(nodeList.Items) == 0 {
		logger.Info("Node not found", "providerID", nodeProviderID)
		return ctrl.Result{}, nil
	}
	node := nodeList.Items[0]
	r.Recorder.AnnotatedEventf(&node,
		map[string]string{
			"host": obj.Spec.InstanceID,
		},
		corev1.EventTypeWarning, "SpotInterrupted",
		"Instance %s is spot interrupted", obj.Spec.InstanceID)

	var podList corev1.PodList
	if err := r.List(ctx, &podList, client.MatchingFields{podNodeNameField: node.Name}); err != nil {
		return ctrl.Result{}, err
	}
	if len(podList.Items) == 0 {
		logger.Info("No pod is affected", "providerID", nodeProviderID, "node", node.Name)
		return ctrl.Result{}, nil
	}
	for _, pod := range podList.Items {
		r.Recorder.AnnotatedEventf(&pod,
			map[string]string{
				"host": obj.Spec.InstanceID,
			},
			corev1.EventTypeWarning, "SpotInterrupted",
			"Instance %s is spot interrupted", obj.Spec.InstanceID)
		//if err := r.Delete(ctx, &pod); err != nil {
		//	return err
		//}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SpotInterruptionReconciler) SetupWithManager(mgr ctrl.Manager) error {
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
		For(&spothandlerv1.SpotInterruption{}).
		Complete(r)
}
