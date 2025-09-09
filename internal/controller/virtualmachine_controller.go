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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
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

	"github.com/innabox/cloudkit-operator/api/v1alpha1"
	kubevirtv1 "kubevirt.io/api/core/v1"
)

// NewVMComponentFn is the type of a function that creates a required component
type NewVMComponentFn func(context.Context, *v1alpha1.VirtualMachine) (*appResource, error)

type vmComponent struct {
	name string
	fn   NewVMComponentFn
}

func (r *VirtualMachineReconciler) vmComponents() []vmComponent {
	return []vmComponent{
		{"Namespace", r.newNamespace},
	}
}

// VirtualMachineReconciler reconciles a VirtualMachine object
type VirtualMachineReconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	CreateVMWebhook         string
	DeleteVMWebhook         string
	VirtualMachineNamespace string
	MinimumRequestInterval  time.Duration
}

func NewVirtualMachineReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	createVMWebhook string,
	deleteVMWebhook string,
	virtualMachineNamespace string,
	minimumRequestInterval time.Duration,
) *VirtualMachineReconciler {

	if virtualMachineNamespace == "" {
		virtualMachineNamespace = defaultVirtualMachineNamespace
	}

	return &VirtualMachineReconciler{
		Client:                  client,
		Scheme:                  scheme,
		CreateVMWebhook:         createVMWebhook,
		DeleteVMWebhook:         deleteVMWebhook,
		VirtualMachineNamespace: virtualMachineNamespace,
		MinimumRequestInterval:  minimumRequestInterval,
	}
}

// +kubebuilder:rbac:groups=cloudkit.openshift.io,resources=virtualmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloudkit.openshift.io,resources=virtualmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloudkit.openshift.io,resources=virtualmachines/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *VirtualMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	instance := &v1alpha1.VirtualMachine{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	val, exists := instance.Annotations[cloudkitVirtualMachineManagementStateAnnotation]
	if exists && val == "unmanaged" {
		log.Info("ignoring VirtualMachine due to management-state annotation", "management-state", val)
		return ctrl.Result{}, nil
	}

	log.Info("start reconcile")

	oldstatus := instance.Status.DeepCopy()

	var res ctrl.Result
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		res, err = r.handleUpdate(ctx, req, instance)
	} else {
		res, err = r.handleDelete(ctx, req, instance)
	}

	if err == nil {
		if !equality.Semantic.DeepEqual(instance.Status, oldstatus) {
			log.Info("status requires update")
			if err := r.Status().Update(ctx, instance); err != nil {
				return res, err
			}
		}
	}

	log.Info("end reconcile")
	return res, err
}

func VirtualMachineNamespacePredicate(namespace string) predicate.Predicate {
	return predicate.NewPredicateFuncs(
		func(obj client.Object) bool {
			return obj.GetNamespace() == namespace
		},
	)
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	labelPredicate, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      cloudkitVirtualMachineNameLabel,
				Operator: metav1.LabelSelectorOpExists,
			},
		},
	})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.VirtualMachine{}, builder.WithPredicates(VirtualMachineNamespacePredicate(r.VirtualMachineNamespace))).
		Watches(
			&corev1.Namespace{},
			handler.EnqueueRequestsFromMapFunc(r.mapObjectToVirtualMachine),
			builder.WithPredicates(labelPredicate),
		).
		Complete(r)
}

// mapObjectToVirtualMachine maps an event for a watched object to the associated
// VirtualMachine resource.
func (r *VirtualMachineReconciler) mapObjectToVirtualMachine(ctx context.Context, obj client.Object) []reconcile.Request {
	log := ctrllog.FromContext(ctx)

	virtualMachineName, exists := obj.GetLabels()[cloudkitVirtualMachineNameLabel]
	if !exists {
		return nil
	}

	// Verify that the referenced VirtualMachine exists in this controller's namespace
	// to filter out notifications for resources managed by other controller instances
	virtualMachine := &v1alpha1.VirtualMachine{}
	key := client.ObjectKey{
		Name:      virtualMachineName,
		Namespace: r.VirtualMachineNamespace,
	}
	if err := r.Get(ctx, key, virtualMachine); err != nil {
		// VirtualMachine doesn't exist in our namespace, ignore this notification
		log.V(1).Info("ignoring notification for resource not managed by this controller instance",
			"kind", obj.GetObjectKind().GroupVersionKind().Kind,
			"namespace", obj.GetNamespace(),
			"name", obj.GetName(),
			"virtualmachine", virtualMachineName,
			"controller_namespace", r.VirtualMachineNamespace,
		)
		return nil
	}

	log.Info("mapped change notification",
		"kind", obj.GetObjectKind().GroupVersionKind().Kind,
		"namespace", obj.GetNamespace(),
		"name", obj.GetName(),
		"virtualmachine", virtualMachineName,
	)

	return []reconcile.Request{
		{
			NamespacedName: key,
		},
	}
}

func (r *VirtualMachineReconciler) handleUpdate(ctx context.Context, _ ctrl.Request, instance *v1alpha1.VirtualMachine) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	r.initializeStatusConditions(instance)
	instance.Status.Phase = v1alpha1.VirtualMachinePhaseProgressing

	if controllerutil.AddFinalizer(instance, cloudkitVirtualMachineFinalizer) {
		if err := r.Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	for _, component := range r.vmComponents() {
		log.Info("handling component", "component", component.name)

		resource, err := component.fn(ctx, instance)
		if err != nil {
			log.Error(err, "failed to mutate resource", "component", component.name)
			return ctrl.Result{}, err
		}

		result, err := controllerutil.CreateOrUpdate(ctx, r.Client, resource.object, resource.mutateFn)
		if err != nil {
			log.Error(err, "failed to create or update component", "component", component.name)
			return ctrl.Result{}, err
		}
		switch result {
		case controllerutil.OperationResultCreated:
			log.Info("created component", "component", component.name)
		case controllerutil.OperationResultUpdated:
			log.Info("updated component", "component", component.name)
		}
	}

	instance.SetStatusCondition(v1alpha1.VirtualMachineConditionAccepted, metav1.ConditionTrue, "", "AsExpected")

	ns, err := r.findNamespace(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	if kv, _ := r.findKubeVirtVMs(ctx, instance, ns.GetName()); kv != nil {
		if err := r.handleKubeVirtVM(ctx, instance, kv); err != nil {
			return ctrl.Result{}, err
		}
	}

	if ns != nil {
		instance.SetStatusCondition(v1alpha1.VirtualMachineConditionAvailable, metav1.ConditionTrue, "", "AsExpected")
		instance.Status.Phase = v1alpha1.VirtualMachinePhaseReady
	}

	if url := r.CreateVMWebhook; url != "" {
		val, exists := instance.Annotations[cloudkitVirtualMachineManagementStateAnnotation]
		if exists && val == "manual" {
			log.Info("not triggering create webhook due to management-state annotation", "url", url, "management-state", val)
		} else {
			remainingTime, err := triggerCreateVMWebHook(ctx, url, instance, r.MinimumRequestInterval)
			if err != nil {
				log.Error(err, "failed to trigger webhook", "url", url, "error", err)
				return ctrl.Result{Requeue: true}, nil
			}

			// Verify if we are within the minimum request window
			if remainingTime != 0 {
				log.Info("request is within minimum request window", "url", url)
				return ctrl.Result{RequeueAfter: remainingTime}, nil
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *VirtualMachineReconciler) findNamespace(ctx context.Context, instance *v1alpha1.VirtualMachine) (*corev1.Namespace, error) {
	log := ctrllog.FromContext(ctx)

	var namespaceList corev1.NamespaceList
	if err := r.List(ctx, &namespaceList, labelSelectorFromVirtualMachineInstance(instance)); err != nil {
		log.Error(err, "failed to list namespaces")
		return nil, err
	}

	if len(namespaceList.Items) > 1 {
		return nil, fmt.Errorf("found too many (%d) matching namespaces for %s", len(namespaceList.Items), instance.GetName())
	}

	if len(namespaceList.Items) == 0 {
		return nil, nil
	}

	return &namespaceList.Items[0], nil
}

func (r *VirtualMachineReconciler) handleDelete(ctx context.Context, _ ctrl.Request, instance *v1alpha1.VirtualMachine) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	log.Info("deleting virtualmachine")

	instance.Status.Phase = v1alpha1.VirtualMachinePhaseDeleting

	if !controllerutil.ContainsFinalizer(instance, cloudkitVirtualMachineFinalizer) {
		return ctrl.Result{}, nil
	}

	ns, err := r.findNamespace(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	if ns != nil {
		// Attempt to delete virtual machine via webhook
		log.Info("waiting for virtual machine to delete", "namespace", ns.GetName())
		if url := r.DeleteVMWebhook; url != "" {
			val, exists := instance.Annotations[cloudkitVirtualMachineManagementStateAnnotation]
			if exists && val == "manual" {
				log.Info("not triggering delete webhook due to management-state annotation", "url", url, "management-state", val)
			} else {
				remainingTime, err := triggerDeleteVMWebHook(ctx, url, instance, r.MinimumRequestInterval)
				if err != nil {
					log.Error(err, "failed to trigger webhook", "url", url, "error", err)
					return ctrl.Result{Requeue: true}, nil
				}

				if remainingTime != 0 {
					return ctrl.Result{RequeueAfter: remainingTime}, nil
				}
			}
		}
		return ctrl.Result{}, err
	}

	// Allow kubernetes to delete the virtualmachine
	if controllerutil.RemoveFinalizer(instance, cloudkitVirtualMachineFinalizer) {
		if err := r.Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// initializeStatusConditions initializes the conditions that haven't already been initialized.
func (r *VirtualMachineReconciler) initializeStatusConditions(instance *v1alpha1.VirtualMachine) {
	r.initializeStatusCondition(
		instance,
		v1alpha1.VirtualMachineConditionAccepted,
		metav1.ConditionTrue,
		v1alpha1.ReasonInitialized,
	)
	r.initializeStatusCondition(
		instance,
		v1alpha1.VirtualMachineConditionDeleting,
		metav1.ConditionFalse,
		v1alpha1.ReasonInitialized,
	)
	r.initializeStatusCondition(
		instance,
		v1alpha1.VirtualMachineConditionProgressing,
		metav1.ConditionTrue,
		v1alpha1.ReasonInitialized,
	)
}

// initializeStatusCondition initializes a condition, but only it is not already initialized.
func (r *VirtualMachineReconciler) initializeStatusCondition(instance *v1alpha1.VirtualMachine,
	conditionType v1alpha1.VirtualMachineConditionType, status metav1.ConditionStatus, reason string) {
	if instance.Status.Conditions == nil {
		instance.Status.Conditions = []metav1.Condition{}
	}
	condition := instance.GetStatusCondition(conditionType)
	if condition != nil {
		return
	}
	instance.SetStatusCondition(conditionType, status, reason, "")
}

func (r *VirtualMachineReconciler) findKubeVirtVMs(ctx context.Context, instance *v1alpha1.VirtualMachine, nsName string) (*kubevirtv1.VirtualMachine, error) {
	log := ctrllog.FromContext(ctx)

	var kubeVirtVMList kubevirtv1.VirtualMachineList
	if err := r.List(ctx, &kubeVirtVMList, client.InNamespace(nsName), labelSelectorFromVirtualMachineInstance(instance)); err != nil {
		log.Error(err, "failed to list KubeVirt VMs")
		return nil, err
	}

	if len(kubeVirtVMList.Items) > 1 {
		return nil, fmt.Errorf("found too many (%d) matching KubeVirt VMs for %s", len(kubeVirtVMList.Items), instance.GetName())
	}

	if len(kubeVirtVMList.Items) == 0 {
		return nil, nil
	}

	return &kubeVirtVMList.Items[0], nil
}

func (r *VirtualMachineReconciler) handleKubeVirtVM(ctx context.Context, instance *v1alpha1.VirtualMachine,
	kv *kubevirtv1.VirtualMachine) error {

	log := ctrllog.FromContext(ctx)

	name := kv.GetName()
	instance.SetVirtualMachineReferenceKubeVirtVirtalMachineName(name)
	instance.SetStatusCondition(v1alpha1.VirtualMachineConditionAccepted, metav1.ConditionTrue, "", v1alpha1.ReasonAsExpected)

	if kvVmHasConditionWithStatus(kv, kubevirtv1.VirtualMachineReady, corev1.ConditionTrue) {
		log.Info("KubeVirt virtual machine is ready", "virtualmachine", instance.GetName())
		instance.SetStatusCondition(v1alpha1.VirtualMachineConditionAvailable, metav1.ConditionTrue, "", v1alpha1.ReasonAsExpected)
		instance.Status.Phase = v1alpha1.VirtualMachinePhaseReady
	}

	return nil
}

func kvVmGetCondition(vm *kubevirtv1.VirtualMachine, cond kubevirtv1.VirtualMachineConditionType) *kubevirtv1.VirtualMachineCondition {
	if vm == nil {
		return nil
	}
	for _, c := range vm.Status.Conditions {
		if c.Type == cond {
			return &c
		}
	}
	return nil
}

func kvVmHasConditionWithStatus(vm *kubevirtv1.VirtualMachine, cond kubevirtv1.VirtualMachineConditionType, status corev1.ConditionStatus) bool {
	c := kvVmGetCondition(vm, cond)
	return c != nil && c.Status == status
}
