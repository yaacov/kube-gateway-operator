/*
Copyright 2021 Yaacov Zamir.

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

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	oauthv1 "github.com/openshift/api/oauth/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ocgatev1beta1 "github.com/rh-fieldwork/kube-gateway-operator/api/v1beta1"
)

const gateserverFinalizer = "ocgate.rh-fieldwork.com/finalizer"

// GateServerReconciler reconciles a GateServer object
type GateServerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="route.openshift.io",resources=routes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="route.openshift.io",resources=routes/custom-host,verbs=create;patch
// +kubebuilder:rbac:groups="oauth.openshift.io",resources=oauthclients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="security.openshift.io",resources=securitycontextconstraints,resourceNames=privileged,verbs=use
// +kubebuilder:rbac:groups="ocgate.rh-fieldwork.com",resources=gateservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="ocgate.rh-fieldwork.com",resources=gateservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="ocgate.rh-fieldwork.com",resources=gateservers/finalizers,verbs=update

// In order to grant client users access to resource, the operator it'slef need this access.
// Note: to create a new gate server with admin role, the role of this operator need to be adjusted.
// +kubebuilder:rbac:groups="*",resources="*",verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the GateServer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *GateServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("Reconcile", "gateserver", req.NamespacedName)

	// your logic here

	// Lookup the GateToken instance for this reconcile request
	s := &ocgatev1beta1.GateServer{}
	if err := r.Get(ctx, req.NamespacedName, s); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.Log.Info("GateServer resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		r.Log.Error(err, "Failed to get GateServer.")
		return ctrl.Result{}, err
	}

	// Check if the GateServer instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isGateServerMarkedToBeDeleted := s.GetDeletionTimestamp() != nil
	if isGateServerMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(s, gateserverFinalizer) {
			// Run finalization logic for gateserverFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeGateServer(s); err != nil {
				return ctrl.Result{}, err
			}

			// Remove gateserverFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(s, gateserverFinalizer)
			if err := r.Update(ctx, s); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// If token was created, exit.
	if s.Status.Phase != "" {
		r.Log.Info("Old server", "id", s.Name)
		return ctrl.Result{}, nil
	}

	// Check role
	var err error
	if len(s.Spec.ServiceAccountNonResourceURLs) != 0 && len(s.Spec.ServiceAccountAPIGroups) != 0 {
		err = fmt.Errorf("auth roles can either apply to API resources or non-resource URL paths, but not both")
	}
	if len(s.Spec.ServiceAccountNonResourceURLs) == 0 && len(s.Spec.ServiceAccountAPIGroups) == 0 {
		err = fmt.Errorf("auth roles can either apply to API resources or non-resource URL paths, but can't be empty")
	}

	if err != nil {
		r.Log.Info("Failed to create oc gate proxy.", "err", err)

		setServerCondition(s, "FailedCreateServer", err)
		if err := r.Status().Update(ctx, s); err != nil {
			r.Log.Info("Failed to update status", "err", err)
		}
		return ctrl.Result{}, nil
	}

	// Create the server
	if err := r.buildServer(ctx, s); err != nil {
		r.Log.Info("Failed to create oc gate proxy.", "err", err)

		setServerCondition(s, "FailedCreateServer", err)
		if err := r.Status().Update(ctx, s); err != nil {
			r.Log.Info("Failed to update status", "err", err)
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(s, gateserverFinalizer) {
		controllerutil.AddFinalizer(s, gateserverFinalizer)
		if err := r.Update(ctx, s); err != nil {
			r.Log.Info("Failed to add finalizer", "err", err)
			return ctrl.Result{}, nil
		}
	}

	// Set status to Ready
	now := metav1.Time{Time: time.Now()}
	s.Status.Phase = "Ready"
	condition := metav1.Condition{
		Type:               "Created",
		Status:             "True",
		Reason:             "AllResourcesCreated",
		Message:            "All resources created",
		LastTransitionTime: now,
	}
	s.Status.Conditions = append(s.Status.Conditions, condition)
	if err := r.Status().Update(ctx, s); err != nil {
		r.Log.Info("Failed to update status", "err", err)
	}

	return ctrl.Result{}, nil
}

func (r *GateServerReconciler) finalizeGateServer(s *ocgatev1beta1.GateServer) error {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.

	ctx := context.Background()
	opts := &client.DeleteOptions{}
	errs := []error{}

	r.Log.Info("Deleting cluster role and cluster role binding...")

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.Name,
		},
	}
	if err := r.Delete(ctx, clusterRole, opts); err != nil {
		r.Log.Info("Failed to finalize clusterRole", "err", err)
		errs = append(errs, err)
	}

	if s.Spec.ServiceAccountNamespace == "*" {
		clusterRoleBinding := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: s.Name,
			},
		}
		if err := r.Delete(ctx, clusterRoleBinding, opts); err != nil {
			r.Log.Info("Failed to finalize clusterRoleBinding", "err", err)
			errs = append(errs, err)
		}
	}

	if s.Spec.GenerateOauthClient {
		r.Log.Info("Deleting oauthclient...")
		oauthclient := &oauthv1.OAuthClient{
			ObjectMeta: metav1.ObjectMeta{
				Name: s.Name,
			},
		}
		if err := r.Delete(ctx, oauthclient, opts); err != nil {
			r.Log.Info("Failed to finalize oauthclient", "err", err)
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		r.Log.Info("Failed to finalized gateserver")
	} else {
		r.Log.Info("Successfully finalized gateserver")
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GateServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ocgatev1beta1.GateServer{}).
		Complete(r)
}
