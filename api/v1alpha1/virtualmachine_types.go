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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VirtualMachineSpec defines the desired state of VirtualMachine
type VirtualMachineSpec struct {
	// TemplateID is the unique identigier of the virtual machine template to use when creating this virtual machine
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern=^[a-zA-Z_][a-zA-Z0-9._]*$
	TemplateID string `json:"templateID,omitempty"`
	// TemplateParameters is a JSON-encoded map of the parameter values for the
	// selected virtual machine template.
	// +kubebuilder:validation:Optional
	TemplateParameters string `json:"templateParameters,omitempty"`
}

// VirtualMachinePhaseType is a valid value for .status.phase
type VirtualMachinePhaseType string

const (
	// VirtualMachinePhaseProgressing means an update is in progress
	VirtualMachinePhaseProgressing VirtualMachinePhaseType = "Progressing"

	// VirtualMachinePhaseFailed means the virtual machine deployment or update has failed
	VirtualMachinePhaseFailed VirtualMachinePhaseType = "Failed"

	// VirtualMachinePhaseReady means the virtual machine and all associated resources are ready
	VirtualMachinePhaseReady VirtualMachinePhaseType = "Ready"

	// VirtualMachinePhaseDeleting means there has been a request to delete the VirtualMachine
	VirtualMachinePhaseDeleting VirtualMachinePhaseType = "Deleting"
)

// VirtualMachineConditionType is a valid value for .status.conditions.type
type VirtualMachineConditionType string

const (
	// VirtualMachineConditionAccepted means the order has been accepted but work has not yet started
	VirtualMachineConditionAccepted VirtualMachineConditionType = "Accepted"

	// VirtualMachineConditionProgressing means that an update is in progress
	VirtualMachineConditionProgressing VirtualMachineConditionType = "Progressing"

	// VirtualMachineConditionAvailable means the virtual machine is available
	VirtualMachineConditionAvailable VirtualMachineConditionType = "Available"

	// VirtualMachineConditionDeleting means the virtual machine is being deleted
	VirtualMachineConditionDeleting VirtualMachineConditionType = "Deleting"
)

// VirtualMachineReferenceType contains a reference to the resources created by this VirtualMachine
type VirtualMachineReferenceType struct {
	// Namespace that contains the VirtualMachine resources
	Namespace                 string `json:"namespace"`
	KubeVirtVirtalMachineName string `json:"kubeVirtVirtalMachineName"`
	ServiceAccountName        string `json:"serviceAccountName"`
	RoleBindingName           string `json:"roleBindingName"`
}

// VirtualMachineStatus defines the observed state of VirtualMachine.
type VirtualMachineStatus struct {
	// Phase provides a single-value overview of the state of the VirtualMachine
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Enum=Progressing;Failed;Ready;Deleting
	Phase VirtualMachinePhaseType `json:"phase,omitempty"`

	// Conditions holds an array of metav1.Condition that describe the state of the VirtualMachine
	// +kubebuilder:validation:Optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// Reference to the namespace that contains VirtualMachine resources
	// +kubebuilder:validation:Optional
	VirtualMachineReference *VirtualMachineReferenceType `json:"virtualMachineReference,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=vm
// +kubebuilder:printcolumn:name="Template",type=string,JSONPath=`.spec.templateID`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// VirtualMachine is the Schema for the virtualmachines API
type VirtualMachine struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of VirtualMachine
	// +required
	Spec VirtualMachineSpec `json:"spec"`

	// status defines the observed state of VirtualMachine
	// +optional
	Status VirtualMachineStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// VirtualMachineList contains a list of VirtualMachine
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualMachine{}, &VirtualMachineList{})
}
