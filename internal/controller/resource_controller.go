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

	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	terragruntv1alpha1 "github.com/JacobAmar/kube-tf-controller/api/v1alpha1"
)

// ResourceReconciler reconciles a Resource object
type ResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=terragrunt.yaakov.com,resources=resources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=terragrunt.yaakov.com,resources=resources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=terragrunt.yaakov.com,resources=resources/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Resource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.2/pkg/reconcile
func (r *ResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	resource := &terragruntv1alpha1.Resource{}
	err := r.Get(ctx, req.NamespacedName, resource)
	if err != nil && apierrors.IsNotFound(err) {
		log.Info("Terragrunt Resource not found for %s, Creating new one in namespace: %s", resource.Name, resource.Namespace)
		job, err := r.jobForTerragrunt(resource)
		if err != nil {
			log.Error(err, "Failed to create job for: %s, issue with setup owner reference, error is: %v", resource.Name, err)
			return ctrl.Result{}, err
		}
		if err = r.Create(ctx, job); err != nil {
			log.Error(err, "Failed to create terragrunt job for: %s, reason is %v", resource.Name, err)
			return ctrl.Result{}, err
		}
	}
	// Checking resource status condition
	if resource.Status.Conditions == nil || len(resource.Status.Conditions) == 0 {
		meta.SetStatusCondition(&resource.Status.Conditions, metav1.Condition{Type: "Unknown", Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})
		if err := r.Status().Update(ctx, resource); err != nil {
			log.Error(err, "Failed to update Terragrunt Resource status for %s in namespace: %s", resource.Name, resource.Namespace)
			return ctrl.Result{}, err
		}
		if err := r.Get(ctx, req.NamespacedName, resource); err != nil {
			log.Error(err, "Failed to re-fetch Terragrunt Resource: %s in namespace: %s", resource.Name, resource.Namespace)
			return ctrl.Result{}, err
		}
	}

	// Checking if the job is already running
	found := &batchv1.Job{}
	err = r.Get(ctx, types.NamespacedName{Name: resource.Name, Namespace: resource.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new job
		job, err := r.jobForTerragrunt(resource)
		if err != nil {
			log.Error(err, "Failed to create job for: %s, in namespace: %s, error is: %v", resource.Name, resource.Namespace, err)
			return ctrl.Result{}, err
		}
		log.Info("Creating a new Job", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
		if err = r.Create(ctx, job); err != nil {
			log.Error(err, "Failed to create new Job", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
			return ctrl.Result{}, err
		}
		// Requeue Reconiliation
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&terragruntv1alpha1.Resource{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

func (r *ResourceReconciler) jobForTerragrunt(m *terragruntv1alpha1.Resource) (*batchv1.Job, error) {
	// Define a new job for terragrunt resource
	path := m.Spec.Path
	// Testing
	image := "alpine/terragrunt:1.10.3"
	command := []string{"echo", "Path: ", path, "&&", "sleep", "3600"}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-terragrunt-%s", m.Name, path),
			Namespace: m.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("%s-terragrunt-%s", m.Name, path),
					Labels: map[string]string{
						"app": fmt.Sprintf("%s-terragunt-%s", m.Name, path),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "terragrunt-runner",
						Image:           image,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Command:         command,
					}},
				},
			},
		},
	}
	if err := ctrl.SetControllerReference(m, job, r.Scheme); err != nil {
		return nil, err
	}
	return job, nil
}
