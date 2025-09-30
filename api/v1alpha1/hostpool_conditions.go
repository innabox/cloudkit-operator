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

// SetCondition sets or updates the condition of the specified type in the HostPool status.
// If a condition of the specified type already exists, its status, reason, and message will be updated.
// If not, a new condition will be added to the list.
func (hp *HostPool) SetCondition(conditionType HostPoolConditionType, status metav1.ConditionStatus, reason, message string) {
	condition := metav1.Condition{
		Type:               string(conditionType),
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	// Find and update existing condition, or append new one
	for i, existingCondition := range hp.Status.Conditions {
		if existingCondition.Type == string(conditionType) {
			if existingCondition.Status != status {
				condition.LastTransitionTime = metav1.Now()
			} else {
				condition.LastTransitionTime = existingCondition.LastTransitionTime
			}
			hp.Status.Conditions[i] = condition
			return
		}
	}

	// If we reach here, the condition doesn't exist, so append it
	hp.Status.Conditions = append(hp.Status.Conditions, condition)
}

// GetCondition returns the condition of the specified type from the HostPool status.
// Returns nil if the condition is not found.
func (hp *HostPool) GetCondition(conditionType HostPoolConditionType) *metav1.Condition {
	for _, condition := range hp.Status.Conditions {
		if condition.Type == string(conditionType) {
			return &condition
		}
	}
	return nil
}

// IsConditionTrue returns true if the condition of the specified type is set to True.
func (hp *HostPool) IsConditionTrue(conditionType HostPoolConditionType) bool {
	condition := hp.GetCondition(conditionType)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsConditionFalse returns true if the condition of the specified type is set to False.
func (hp *HostPool) IsConditionFalse(conditionType HostPoolConditionType) bool {
	condition := hp.GetCondition(conditionType)
	return condition != nil && condition.Status == metav1.ConditionFalse
}

// RemoveCondition removes the condition of the specified type from the HostPool status.
func (hp *HostPool) RemoveCondition(conditionType HostPoolConditionType) {
	for i, condition := range hp.Status.Conditions {
		if condition.Type == string(conditionType) {
			hp.Status.Conditions = append(hp.Status.Conditions[:i], hp.Status.Conditions[i+1:]...)
			return
		}
	}
}
