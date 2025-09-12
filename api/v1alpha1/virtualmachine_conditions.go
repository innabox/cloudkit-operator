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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Additional reason constants for VirtualMachine conditions
const (
	ReasonFailed           = "Failed"
	ReasonDeleting         = "Deleting"
	ReasonWebhookTriggered = "WebhookTriggered"
	ReasonWebhookFailed    = "WebhookFailed"
)

// SetStatusCondition sets the condition on the VirtualMachine status
func (vm *VirtualMachine) SetStatusCondition(conditionType VirtualMachineConditionType, status metav1.ConditionStatus, reason, message string) {
	condition := metav1.Condition{
		Type:               string(conditionType),
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}

	// Initialize conditions slice if nil
	if vm.Status.Conditions == nil {
		vm.Status.Conditions = []metav1.Condition{}
	}

	// Find existing condition and update or add new one
	found := false
	for i, existing := range vm.Status.Conditions {
		if existing.Type == string(conditionType) {
			vm.Status.Conditions[i] = condition
			found = true
			break
		}
	}

	if !found {
		vm.Status.Conditions = append(vm.Status.Conditions, condition)
	}
}

// GetStatusCondition returns the condition with the given type
func (vm *VirtualMachine) GetStatusCondition(conditionType VirtualMachineConditionType) *metav1.Condition {
	if vm.Status.Conditions == nil {
		return nil
	}

	for i := range vm.Status.Conditions {
		if vm.Status.Conditions[i].Type == string(conditionType) {
			return &vm.Status.Conditions[i]
		}
	}
	return nil
}

// IsStatusConditionTrue returns true if the condition with the given type is true
func (vm *VirtualMachine) IsStatusConditionTrue(conditionType VirtualMachineConditionType) bool {
	condition := vm.GetStatusCondition(conditionType)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsStatusConditionFalse returns true if the condition with the given type is false
func (vm *VirtualMachine) IsStatusConditionFalse(conditionType VirtualMachineConditionType) bool {
	condition := vm.GetStatusCondition(conditionType)
	return condition != nil && condition.Status == metav1.ConditionFalse
}

// IsStatusConditionUnknown returns true if the condition with the given type is unknown
func (vm *VirtualMachine) IsStatusConditionUnknown(conditionType VirtualMachineConditionType) bool {
	condition := vm.GetStatusCondition(conditionType)
	return condition == nil || condition.Status == metav1.ConditionUnknown
}
