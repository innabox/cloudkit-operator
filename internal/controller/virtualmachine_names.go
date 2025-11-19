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
	"fmt"

	v1alpha1 "github.com/innabox/cloudkit-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/rand"
)

const (
	defaultVirtualMachineNamespace string = "cloudkit-vm-orders"
	cloudkitVMNamePrefix           string = "cloudkit.openshift.io"
	cloudkitAAPVMNamePrefix        string = "cloudkit-aap.openshift.io"
)

var (
	cloudkitVirtualMachineNameLabel                  string = fmt.Sprintf("%s/virtualmachine", cloudkitVMNamePrefix)
	cloudkitVirtualMachineIDLabel                    string = fmt.Sprintf("%s/virtualmachine-uuid", cloudkitVMNamePrefix)
	cloudkitVirtualMachineFinalizer                  string = fmt.Sprintf("%s/finalizer", cloudkitVMNamePrefix)
	cloudkitAAPVirtualMachineFinalizer               string = fmt.Sprintf("%s/finalizer", cloudkitAAPVMNamePrefix)
	cloudkitVirtualMachineManagementStateAnnotation  string = fmt.Sprintf("%s/management-state", cloudkitVMNamePrefix)
	cloudkitVirualMachineFloatingIPAddressAnnotation string = fmt.Sprintf("%s/floating-ip-address", cloudkitVMNamePrefix)
	cloudkitAAPReconciledConfigVersionAnnotation     string = fmt.Sprintf("%s/reconciled-config-version", cloudkitAAPVMNamePrefix)
)

func generateVirtualMachineNamespaceName(instance *v1alpha1.VirtualMachine) string {
	return fmt.Sprintf("vm-%s-%s", instance.GetName(), rand.String(6))
}
