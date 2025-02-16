/*
Copyright 2025.

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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
)

// SpotInterruptedPodTerminationReconciler reconciles a SpotInterruptedPodTermination object
type SpotInterruptedPodTerminationReconciler struct {
	ctrlclient.Client
	Scheme *runtime.Scheme
	Clock  clock.PassiveClock
}

// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=spotinterruptedpodterminations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=spotinterruptedpodterminations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=spotinterruptedpodterminations/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SpotInterruptedPodTerminationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrllog.FromContext(ctx)

	var obj spothandlerv1.SpotInterruptedPodTermination
	if err := r.Get(ctx, req.NamespacedName, &obj); err != nil {
		return ctrl.Result{}, ctrlclient.IgnoreNotFound(err)
	}
	if !obj.Status.ReconciledAt.IsZero() {
		return ctrl.Result{}, nil
	}

	if obj.Spec.TerminationTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}
	timeUntilPodTermination := obj.Spec.TerminationTimestamp.Sub(r.Clock.Now())
	if timeUntilPodTermination > 0 {
		logger.Info("Requeue after the termination timestamp", "timeUntilPodTermination", timeUntilPodTermination, "terminationTimestamp", obj.Spec.TerminationTimestamp)
		return ctrl.Result{RequeueAfter: timeUntilPodTermination}, nil
	}

	if err := r.reconcile(ctx, &obj); err != nil {
		return ctrl.Result{}, err
	}
	obj.Status.ReconciledAt = metav1.NewTime(r.Clock.Now())
	if err := r.Status().Update(ctx, &obj); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update the status of SpotInterruptedPodTermination: %w", err)
	}
	logger.Info("Successfully reconciled SpotInterruptedPodTermination")
	return ctrl.Result{}, nil
}

func (r *SpotInterruptedPodTerminationReconciler) reconcile(ctx context.Context, obj *spothandlerv1.SpotInterruptedPodTermination) error {
	var pod corev1.Pod
	if err := r.Get(ctx, ctrlclient.ObjectKey{Name: obj.Spec.Pod.Name, Namespace: obj.Namespace}, &pod); err != nil {
		return ctrlclient.IgnoreNotFound(fmt.Errorf("failed to get the Pod: %w", err))
	}
	isDaemonPod := slices.ContainsFunc(pod.OwnerReferences, func(owner metav1.OwnerReference) bool {
		return owner.APIVersion == "apps/v1" && owner.Kind == "DaemonSet"
	})
	if isDaemonPod {
		return nil
	}

	var deleteOpts []ctrlclient.DeleteOption
	if obj.Spec.GracePeriodSeconds != nil {
		deleteOpts = append(deleteOpts, ctrlclient.GracePeriodSeconds(*obj.Spec.GracePeriodSeconds))
	}
	obj.Status.RequestedAt = metav1.NewTime(r.Clock.Now())
	if err := r.Delete(ctx, &pod, deleteOpts...); err != nil {
		obj.Status.RequestError = fmt.Sprintf("delete error: %s", err)
	}
	if err := r.createPodTerminatingEvent(ctx, *obj, pod); err != nil {
		return err
	}
	return nil
}

func (r *SpotInterruptedPodTerminationReconciler) createPodTerminatingEvent(ctx context.Context, obj spothandlerv1.SpotInterruptedPodTermination, pod corev1.Pod) error {
	logger := ctrllog.FromContext(ctx)

	var message string
	switch {
	case obj.Status.RequestError != "":
		message = fmt.Sprintf("Failed to terminate the Pod %s on Node %s of %s: %s",
			pod.Name, obj.Spec.Node.Name, obj.Spec.InstanceID, obj.Status.RequestError)
	case obj.Spec.GracePeriodSeconds != nil:
		message = fmt.Sprintf("Pod %s on Node %s of %s is terminating with grace period %d seconds.",
			pod.Name, obj.Spec.Node.Name, obj.Spec.InstanceID, *obj.Spec.GracePeriodSeconds)
	default:
		message = fmt.Sprintf("Pod %s on Node %s of %s is terminating.", pod.Name, obj.Spec.Node.Name, obj.Spec.InstanceID)
	}

	ref, err := reference.GetReference(r.Scheme, &pod)
	if err != nil {
		return fmt.Errorf("failed to get the reference of the Pod: %w", err)
	}
	// We emit an event without the EventRecorder because:
	//  - Set the Host field.
	//  - Emit an event exactly once.
	source := corev1.EventSource{
		Component: "spotinterruptedpodtermination-controller",
		Host:      obj.Spec.Node.Name,
	}
	t := metav1.NewTime(r.Clock.Now())
	event := corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("podterminating-%s", obj.Name),
			Namespace: obj.Namespace,
		},
		Source:              source,
		ReportingController: source.Component,
		ReportingInstance:   source.Host,
		InvolvedObject:      *ref,
		FirstTimestamp:      t,
		LastTimestamp:       t,
		Count:               1,
		Type:                corev1.EventTypeNormal,
		Reason:              "PodTerminating",
		Message:             message,
	}
	if err := r.Create(ctx, &event); err != nil {
		return ctrlclient.IgnoreAlreadyExists(fmt.Errorf("failed to create Event: %w", err))
	}
	logger.Info("Created an Event", "reason", event.Reason, "message", event.Message)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SpotInterruptedPodTerminationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&spothandlerv1.SpotInterruptedPodTermination{}).
		Named("spotinterruptedpodtermination").
		Complete(r)
}
