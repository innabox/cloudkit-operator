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

// Package controller implements the controller logic
package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cloudkitv1alpha1 "github.com/innabox/cloudkit-operator/api/v1alpha1"
)

// NewComponentFn is the type of a function that creates a required component
type NewComponentFn func(context.Context, *cloudkitv1alpha1.ClusterOrder) (*appResource, error)

type appResource struct {
	object      client.Object
	mutateFn    controllerutil.MutateFn
	shouldExist bool
}

type component struct {
	name string
	fn   NewComponentFn
}

func (r *ClusterOrderReconciler) components() []component {
	return []component{
		{"Namespace", r.newNamespace},
		{"ServiceAccount", r.newServiceAccount},
		{"RoleBinding", r.newAdminRoleBinding},
	}
}

// ClusterOrderReconciler reconciles a ClusterOrder object
type ClusterOrderReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func newClusterReference() *cloudkitv1alpha1.ClusterOrderClusterReferenceType {
	return &cloudkitv1alpha1.ClusterOrderClusterReferenceType{}
}

// +kubebuilder:rbac:groups=cloudkit.openshift.io,resources=clusterorders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloudkit.openshift.io,resources=clusterorders/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloudkit.openshift.io,resources=clusterorders/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ClusterOrderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = ctrllog.FromContext(ctx)

	instance := &cloudkitv1alpha1.ClusterOrder{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleUpdate(ctx, req, instance)
	} else {
		return r.handleDelete(ctx, req, instance)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterOrderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	labelPredicate, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{
		MatchLabels: map[string]string{
			cloudkitClusterOrderNameLabel: "",
		},
	})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudkitv1alpha1.ClusterOrder{}).
		Watches(
			&corev1.Namespace{},
			handler.EnqueueRequestsFromMapFunc(r.mapObjectToCluster),
			builder.WithPredicates(labelPredicate),
		).
		Watches(
			&corev1.ServiceAccount{},
			handler.EnqueueRequestsFromMapFunc(r.mapObjectToCluster),
			builder.WithPredicates(labelPredicate),
		).
		Watches(
			&rbacv1.RoleBinding{},
			handler.EnqueueRequestsFromMapFunc(r.mapObjectToCluster),
			builder.WithPredicates(labelPredicate),
		).
		Complete(r)
}

func (r *ClusterOrderReconciler) mapObjectToCluster(ctx context.Context, obj client.Object) []reconcile.Request {
	log := ctrllog.FromContext(ctx)

	clusterOrderName, exists := obj.GetLabels()[cloudkitClusterOrderNameLabel]
	if !exists {
		return nil
	}

	clusterOrderNamespace, exists := obj.GetLabels()[cloudkitClusterOrderNamespaceLabel]
	if !exists {
		return nil
	}

	log.Info("Selecting " + obj.GetName())

	return []reconcile.Request{
		{
			NamespacedName: client.ObjectKey{
				Name:      clusterOrderName,
				Namespace: clusterOrderNamespace,
			},
		},
	}
}

func (r *ClusterOrderReconciler) handleUpdate(ctx context.Context, _ ctrl.Request, instance *cloudkitv1alpha1.ClusterOrder) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	log.Info("Create or update " + instance.GetName())

	// Do we have a finalizer yet?
	if !controllerutil.ContainsFinalizer(instance, cloudkitFinalizer) {
		controllerutil.AddFinalizer(instance, cloudkitFinalizer)
		if err := r.Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Do we have a clusterReference yet?
	if instance.Status.ClusterReference == nil {
		log.Info("Adding cluster reference")
		instance.Status.ClusterReference = newClusterReference()
		if err := r.Status().Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	for _, component := range r.components() {
		log.Info("Handling component " + component.name)

		resource, err := component.fn(ctx, instance)
		if err != nil {
			log.Error(err, "Failed to mutate resource", component.name)
			return ctrl.Result{}, err
		}

		if resource.shouldExist {
			result, err := controllerutil.CreateOrUpdate(ctx, r.Client, resource.object, resource.mutateFn)
			if err != nil {
				log.Error(err, "Failed to create or update "+component.name)
				return ctrl.Result{}, err
			}
			switch result {
			case controllerutil.OperationResultCreated:
				log.Info("Created " + component.name)
			case controllerutil.OperationResultUpdated:
				log.Info("Updated " + component.name)
			}

			// apply any updates to status.clusterReference
			if err := r.Status().Update(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}

			continue
		}

		// If we get this far, resource should not exist
		// Ensure the resource does not exist, and call Delete if necessary
		key := client.ObjectKeyFromObject(resource.object)
		if err := r.Client.Get(ctx, key, resource.object); err != nil {
			if apierrors.IsNotFound(err) {
				// "not found" is the desired state. Nothing else to do.
				continue
			}
			log.Error(err, "Get request for resource failed", component.name)
			return ctrl.Result{}, err
		}
		err = r.Client.Delete(ctx, resource.object)
		if err != nil {
			log.Error(err, "Delete request for resource failed", component.name)
			return ctrl.Result{}, err
		}
		log.Info("Deleted " + component.name)
	}

	return ctrl.Result{}, nil
}

func (r *ClusterOrderReconciler) handleDelete(ctx context.Context, _ ctrl.Request, instance *cloudkitv1alpha1.ClusterOrder) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	log.Info("Delete " + instance.GetName())

	if controllerutil.ContainsFinalizer(instance, cloudkitFinalizer) {
		labelSelector := client.MatchingLabels{cloudkitClusterOrderNameLabel: instance.GetName()}

		var namespaceList corev1.NamespaceList
		if err := r.List(ctx, &namespaceList, labelSelector); err != nil {
			log.Error(err, "Failed to list Namespaces")
			return ctrl.Result{}, err
		}

		if len(namespaceList.Items) == 0 {
			controllerutil.RemoveFinalizer(instance, cloudkitFinalizer)
			if err := r.Update(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}

		for _, ns := range namespaceList.Items {
			if err := r.Client.Delete(ctx, &ns); err != nil {
				log.Error(err, "Failed to delete namespace "+ns.GetName())
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}
