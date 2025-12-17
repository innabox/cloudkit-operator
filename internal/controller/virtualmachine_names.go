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
)

const (
	defaultVirtualMachineNamespace string = "cloudkit-vm-orders"
)

var (
	cloudkitVirtualMachineNameLabel                  string = fmt.Sprintf("%s/virtualmachine", cloudkitPrefix)
	cloudkitVirtualMachineIDLabel                    string = fmt.Sprintf("%s/virtualmachine-uuid", cloudkitPrefix)
	cloudkitVirtualMachineFinalizer                  string = fmt.Sprintf("%s/virtualmachine", cloudkitPrefix)
	cloudkitAAPVirtualMachineFinalizer               string = fmt.Sprintf("%s/virtualmachine-aap", cloudkitPrefix)
	cloudkitVirtualMachineManagementStateAnnotation  string = fmt.Sprintf("%s/management-state", cloudkitPrefix)
	cloudkitVirualMachineFloatingIPAddressAnnotation string = fmt.Sprintf("%s/floating-ip-address", cloudkitPrefix)
	cloudkitAAPReconciledConfigVersionAnnotation     string = fmt.Sprintf("%s/reconciled-config-version", cloudkitPrefix)
)
