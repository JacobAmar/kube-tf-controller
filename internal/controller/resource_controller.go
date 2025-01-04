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

	appsv1 "k8s.io/api/apps/v1"
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
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Terragrunt Resource not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Terragrunt Resource")
		return ctrl.Result{}, err
	}
	// Checking resource status condition
	if resource.Status.Conditions == nil || len(resource.Status.Conditions) == 0 {
		meta.SetStatusCondition(&resource.Status.Conditions, metav1.Condition{Type: "OutOfSync", Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})
		if err := r.Status().Update(ctx, resource); err != nil {
			log.Error(err, "Failed to update Terragrunt Resource status")
			return ctrl.Result{}, err
		}
		if err := r.Get(ctx, req.NamespacedName, resource); err != nil {
			log.Error(err, "Failed to re-fetch Terragrunt Resource")
			return ctrl.Result{}, err
		}
	}

	// Checking if the deployment is already running
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: resource.Name, Namespace: resource.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new deployment
		dep, err := r.deploymentForTerragrunt(resource)
		if err != nil {
			log.Error(err, "Failed to create deployment")
			return ctrl.Result{}, err
		}
		log.Info("Creating a new Deployment",
			"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		if err = r.Create(ctx, dep); err != nil {
			log.Error(err, "Failed to create new Deployment",
				"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
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

func (r *ResourceReconciler) deploymentForTerragrunt(m *terragruntv1alpha1.Resource) (*appsv1.Deployment, error) {
	// Define a new deployment for terragrunt resource
	path := m.Spec.Path
	replicas := int32(1)
	// Testing
	image := "ubuntu:22.04"
	command := []string{"echo", "Path: ", path, "&&", "sleep", "3600"}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-terragrunt-%s", m.Name, path),
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": fmt.Sprintf("%s-terragrunt-%s", m.Name, path),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": fmt.Sprintf("%s-terragunt-%s", m.Name, path),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            fmt.Sprintf("%s-terragunt-%s", m.Name, path),
						Image:           image,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Command:         command,
					}},
				},
			},
		},
	}
	if err := ctrl.SetControllerReference(m, deployment, r.Scheme); err != nil {
		return nil, err
	}
	return deployment, nil
}
