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

// SpotInterruptedPodReconciler reconciles a SpotInterruptedPod object
type SpotInterruptedPodReconciler struct {
	ctrlclient.Client
	Scheme *runtime.Scheme
	Clock  clock.PassiveClock
}

// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=spotinterruptedpods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=spotinterruptedpods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=spotinterruptedpods/finalizers,verbs=update

// +kubebuilder:rbac:groups=spothandler.int128.github.io,resources=podpolicies,verbs=get;list;watch

// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SpotInterruptedPodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrllog.FromContext(ctx)

	var obj spothandlerv1.SpotInterruptedPod
	if err := r.Get(ctx, req.NamespacedName, &obj); err != nil {
		return ctrl.Result{}, ctrlclient.IgnoreNotFound(err)
	}
	if !obj.Status.ReconciledAt.IsZero() {
		return ctrl.Result{}, nil
	}

	if err := r.reconcile(ctx, &obj); err != nil {
		return ctrl.Result{}, err
	}
	obj.Status.ReconciledAt = metav1.NewTime(r.Clock.Now())
	if err := r.Status().Update(ctx, &obj); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update the status of SpotInterruptedPod: %w", err)
	}
	logger.Info("Successfully reconciled SpotInterruptedPod")
	return ctrl.Result{}, nil
}

func (r *SpotInterruptedPodReconciler) reconcile(ctx context.Context, obj *spothandlerv1.SpotInterruptedPod) error {
	var pod corev1.Pod
	if err := r.Get(ctx, ctrlclient.ObjectKey{Name: obj.Spec.Pod.Name, Namespace: obj.Namespace}, &pod); err != nil {
		return ctrlclient.IgnoreNotFound(fmt.Errorf("failed to get the Pod: %w", err))
	}
	terminatedByPodPolicy, err := r.terminatePodByPolicy(ctx, pod)
	if err != nil {
		return err
	}
	obj.Status.TerminatedByPodPolicy = terminatedByPodPolicy
	if err := r.createEvent(ctx, *obj, pod); err != nil {
		return err
	}
	return nil
}

func (r *SpotInterruptedPodReconciler) terminatePodByPolicy(ctx context.Context, pod corev1.Pod) (bool, error) {
	logger := ctrllog.FromContext(ctx)

	isDaemonPod := slices.ContainsFunc(pod.OwnerReferences, func(owner metav1.OwnerReference) bool {
		return owner.APIVersion == "apps/v1" && owner.Kind == "DaemonSet"
	})
	if isDaemonPod {
		return false, nil
	}
	var podPolicyList spothandlerv1.PodPolicyList
	if err := r.List(ctx, &podPolicyList); err != nil {
		return false, fmt.Errorf("failed to list PodPolicy: %w", err)
	}
	terminateByPolicy := slices.ContainsFunc(podPolicyList.Items, func(podPolicy spothandlerv1.PodPolicy) bool {
		return podPolicy.Spec.TerminateOnSpotInterruption
	})
	if !terminateByPolicy {
		return false, nil
	}
	if err := r.Delete(ctx, &pod); err != nil {
		return false, ctrlclient.IgnoreNotFound(fmt.Errorf("failed to delete the Pod: %w", err))
	}
	logger.Info("Terminating the Pod due to the PodPolicy", "pod", pod.Name)
	return true, nil
}

func (r *SpotInterruptedPodReconciler) createEvent(ctx context.Context, obj spothandlerv1.SpotInterruptedPod, pod corev1.Pod) error {
	logger := ctrllog.FromContext(ctx)

	eventMessage := fmt.Sprintf("Pod %s on Node %s of %s is interrupted.", pod.Name, obj.Spec.Node.Name, obj.Spec.InstanceID)
	if obj.Status.TerminatedByPodPolicy {
		eventMessage += " Pod is terminated by the PodPolicy."
	}
	ref, err := reference.GetReference(r.Scheme, &pod)
	if err != nil {
		return fmt.Errorf("failed to get the reference of the Pod: %w", err)
	}
	// We emit an event without the EventRecorder because:
	//  - Set the Host field.
	//  - Emit an event exactly once.
	source := corev1.EventSource{
		Component: "spotinterruptedpod-controller",
		Host:      obj.Spec.Node.Name,
	}
	t := metav1.NewTime(r.Clock.Now())
	event := corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("spotinterruptedpod-%s", obj.Name),
			Namespace: obj.Namespace,
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
		Message:             eventMessage,
	}
	if err := r.Create(ctx, &event); err != nil {
		return ctrlclient.IgnoreAlreadyExists(fmt.Errorf("failed to create Event: %w", err))
	}
	logger.Info("Created an Event", "reason", event.Reason, "message", event.Message)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SpotInterruptedPodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&spothandlerv1.SpotInterruptedPod{}).
		Complete(r)
}
