/*
Copyright (c) 2025 Red Hat Inc.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the
License. You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific
language governing permissions and limitations under the License.
*/

package controller

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	clnt "sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	ckv1alpha1 "github.com/innabox/cloudkit-operator/api/v1alpha1"
	privatev1 "github.com/innabox/cloudkit-operator/internal/api/private/v1"
	sharedv1 "github.com/innabox/cloudkit-operator/internal/api/shared/v1"
)

// VirtualMachineFeedbackReconciler sends updates to the fulfillment service.
type VirtualMachineFeedbackReconciler struct {
	hubClient               clnt.Client
	virtualMachinesClient   privatev1.VirtualMachinesClient
	virtualMachineNamespace string
}

// virtualMachineFeedbackReconcilerTask contains data that is used for the reconciliation of a specific virtual machine, so there is less
// need to pass around as function parameters that and other related objects.
type virtualMachineFeedbackReconcilerTask struct {
	r      *VirtualMachineFeedbackReconciler
	object *ckv1alpha1.VirtualMachine
	vm     *privatev1.VirtualMachine
}

// NewVirtualMachineFeedbackReconciler creates a reconciler that sends to the fulfillment service updates about virtual machines.
func NewVirtualMachineFeedbackReconciler(hubClient clnt.Client, grpcConn *grpc.ClientConn, virtualMachineNamespace string) *VirtualMachineFeedbackReconciler {
	return &VirtualMachineFeedbackReconciler{
		hubClient:               hubClient,
		virtualMachinesClient:   privatev1.NewVirtualMachinesClient(grpcConn),
		virtualMachineNamespace: virtualMachineNamespace,
	}
}

// SetupWithManager adds the reconciler to the controller manager.
func (r *VirtualMachineFeedbackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("virtualmachine-feedback").
		For(&ckv1alpha1.VirtualMachine{}, builder.WithPredicates(VirtualMachineNamespacePredicate(r.virtualMachineNamespace))).
		Complete(r)
}

// Reconcile is the implementation of the reconciler interface.
func (r *VirtualMachineFeedbackReconciler) Reconcile(ctx context.Context, request ctrl.Request) (result ctrl.Result, err error) {
	log := ctrllog.FromContext(ctx)

	// Fetch the object to reconcile, and do nothing if it no longer exists:
	object := &ckv1alpha1.VirtualMachine{}
	err = r.hubClient.Get(ctx, request.NamespacedName, object)
	if err != nil {
		err = clnt.IgnoreNotFound(err)
		return //nolint:nakedret
	}

	// Get the identifier of the virtual machine from the labels. If this isn't present it means that the object wasn't
	// created by the fulfillment service, so we ignore it.
	vmID, ok := object.Labels[cloudkitVirtualMachineIDLabel]
	if !ok {
		log.Info(
			"There is no label containing the virtual machine identifier, will ignore it",
			"label", cloudkitVirtualMachineIDLabel,
		)
		return
	}

	// Check if the VM is being deleted before fetching from fulfillment service
	if !object.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Info(
			"VirtualMachine is being deleted, skipping feedback reconciliation",
		)
		return
	}

	// Fetch the virtual machine:
	vm, err := r.fetchVirtualMachine(ctx, vmID)
	if err != nil {
		return
	}

	// Create a task to do the rest of the job, but using copies of the objects, so that we can later compare the
	// before and after values and save only the objects that have changed.
	t := &virtualMachineFeedbackReconcilerTask{
		r:      r,
		object: object,
		vm:     clone(vm),
	}

	t.handleUpdate(ctx)

	// Save the objects that have changed:
	err = r.saveVirtualMachine(ctx, vm, t.vm)
	if err != nil {
		return
	}
	return
}

func (r *VirtualMachineFeedbackReconciler) fetchVirtualMachine(ctx context.Context, id string) (vm *privatev1.VirtualMachine, err error) {
	response, err := r.virtualMachinesClient.Get(ctx, privatev1.VirtualMachinesGetRequest_builder{
		Id: id,
	}.Build())
	if err != nil {
		return
	}
	vm = response.GetObject()
	if !vm.HasSpec() {
		vm.SetSpec(&privatev1.VirtualMachineSpec{})
	}
	if !vm.HasStatus() {
		vm.SetStatus(&privatev1.VirtualMachineStatus{})
	}
	return
}

func (r *VirtualMachineFeedbackReconciler) saveVirtualMachine(ctx context.Context, before, after *privatev1.VirtualMachine) error {
	log := ctrllog.FromContext(ctx)

	if !equal(after, before) {
		log.Info(
			"Updating virtual machine",
			"before", before,
			"after", after,
		)
		_, err := r.virtualMachinesClient.Update(ctx, privatev1.VirtualMachinesUpdateRequest_builder{
			Object: after,
		}.Build())
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *virtualMachineFeedbackReconcilerTask) handleUpdate(ctx context.Context) {
	t.syncConditions(ctx)
	t.syncPhase(ctx)
}

func (t *virtualMachineFeedbackReconcilerTask) syncConditions(ctx context.Context) {
	t.syncProgressing(ctx)
	t.syncReady(ctx)
}

// syncProgressing synchronizes the PROGRESSING VM condition from multiple CR conditions.
// If any of Progressing, or Accepted is true, then PROGRESSING is set to true.
func (t *virtualMachineFeedbackReconcilerTask) syncProgressing(ctx context.Context) {
	progressingCondition := t.object.GetStatusCondition(ckv1alpha1.VirtualMachineConditionProgressing)
	acceptedCondition := t.object.GetStatusCondition(ckv1alpha1.VirtualMachineConditionAccepted)

	var newStatus sharedv1.ConditionStatus
	var message string

	if t.object.IsStatusConditionUnknown(ckv1alpha1.VirtualMachineConditionProgressing) && t.object.IsStatusConditionUnknown(ckv1alpha1.VirtualMachineConditionAccepted) {
		newStatus = sharedv1.ConditionStatus_CONDITION_STATUS_UNSPECIFIED
	} else if t.object.IsStatusConditionTrue(ckv1alpha1.VirtualMachineConditionProgressing) {
		newStatus = t.mapConditionStatus(progressingCondition.Status)
		message = progressingCondition.Message
	} else if t.object.IsStatusConditionTrue(ckv1alpha1.VirtualMachineConditionAccepted) {
		newStatus = t.mapConditionStatus(acceptedCondition.Status)
		message = acceptedCondition.Message
	} else {
		newStatus = sharedv1.ConditionStatus_CONDITION_STATUS_FALSE
	}

	vmCondition := t.findVirtualMachineCondition(privatev1.VirtualMachineConditionType_VIRTUAL_MACHINE_CONDITION_TYPE_PROGRESSING)
	oldStatus := vmCondition.GetStatus()

	vmCondition.SetStatus(newStatus)
	vmCondition.SetMessage(message)
	if newStatus != oldStatus {
		vmCondition.SetLastTransitionTime(timestamppb.Now())
	}
}

// syncReady synchronizes the READY VM condition from the Available CR condition.
func (t *virtualMachineFeedbackReconcilerTask) syncReady(ctx context.Context) {
	crCondition := t.object.GetStatusCondition(ckv1alpha1.VirtualMachineConditionAvailable)
	if crCondition == nil {
		return
	}
	t.syncVMConditionFromCR(privatev1.VirtualMachineConditionType_VIRTUAL_MACHINE_CONDITION_TYPE_READY, crCondition)
}

// syncVMConditionFromCR synchronizes a VM condition from a CR condition.
func (t *virtualMachineFeedbackReconcilerTask) syncVMConditionFromCR(vmConditionType privatev1.VirtualMachineConditionType, crCondition *metav1.Condition) {
	vmCondition := t.findVirtualMachineCondition(vmConditionType)
	oldStatus := vmCondition.GetStatus()
	newStatus := t.mapConditionStatus(crCondition.Status)
	vmCondition.SetStatus(newStatus)
	vmCondition.SetMessage(crCondition.Message)
	if newStatus != oldStatus {
		vmCondition.SetLastTransitionTime(timestamppb.Now())
	}
}

func (t *virtualMachineFeedbackReconcilerTask) mapConditionStatus(status metav1.ConditionStatus) sharedv1.ConditionStatus {
	switch status {
	case metav1.ConditionFalse:
		return sharedv1.ConditionStatus_CONDITION_STATUS_FALSE
	case metav1.ConditionTrue:
		return sharedv1.ConditionStatus_CONDITION_STATUS_TRUE
	default:
		return sharedv1.ConditionStatus_CONDITION_STATUS_UNSPECIFIED
	}
}

func (t *virtualMachineFeedbackReconcilerTask) syncPhase(ctx context.Context) {
	switch t.object.Status.Phase {
	case ckv1alpha1.VirtualMachinePhaseProgressing:
		t.syncPhaseProgressing()
	case ckv1alpha1.VirtualMachinePhaseFailed:
		t.syncPhaseFailed()
	case ckv1alpha1.VirtualMachinePhaseReady:
		t.syncPhaseReady()
	default:
		log := ctrllog.FromContext(ctx)
		log.Info(
			"Unknown phase, will ignore it",
			"phase", t.object.Status.Phase,
		)
	}
}

func (t *virtualMachineFeedbackReconcilerTask) syncPhaseProgressing() {
	t.vm.GetStatus().SetState(privatev1.VirtualMachineState_VIRTUAL_MACHINE_STATE_PROGRESSING)
}

func (t *virtualMachineFeedbackReconcilerTask) syncPhaseFailed() {
	t.vm.GetStatus().SetState(privatev1.VirtualMachineState_VIRTUAL_MACHINE_STATE_FAILED)
}

func (t *virtualMachineFeedbackReconcilerTask) syncPhaseReady() {
	vmStatus := t.vm.GetStatus()
	vmStatus.SetState(privatev1.VirtualMachineState_VIRTUAL_MACHINE_STATE_READY)
}

func (t *virtualMachineFeedbackReconcilerTask) findVirtualMachineCondition(kind privatev1.VirtualMachineConditionType) *privatev1.VirtualMachineCondition {
	var condition *privatev1.VirtualMachineCondition
	for _, current := range t.vm.Status.Conditions {
		if current.Type == kind {
			condition = current
			break
		}
	}
	if condition == nil {
		condition = &privatev1.VirtualMachineCondition{
			Type:   kind,
			Status: sharedv1.ConditionStatus_CONDITION_STATUS_FALSE,
		}
		t.vm.Status.Conditions = append(t.vm.Status.Conditions, condition)
	}
	return condition
}
