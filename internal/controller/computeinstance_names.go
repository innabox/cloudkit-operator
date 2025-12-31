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
	defaultComputeInstanceNamespace string = "cloudkit-computeinstance-orders"
)

var (
	cloudkitComputeInstanceNameLabel                 string = fmt.Sprintf("%s/computeinstance", cloudkitPrefix)
	cloudkitComputeInstanceIDLabel                   string = fmt.Sprintf("%s/computeinstance-uuid", cloudkitPrefix)
	cloudkitComputeInstanceFinalizer                 string = fmt.Sprintf("%s/computeinstance", cloudkitPrefix)
	cloudkitAAPComputeInstanceFinalizer              string = fmt.Sprintf("%s/computeinstance-aap", cloudkitPrefix)
	cloudkitComputeInstanceManagementStateAnnotation string = fmt.Sprintf("%s/management-state", cloudkitPrefix)
	cloudkitVirualMachineFloatingIPAddressAnnotation string = fmt.Sprintf("%s/floating-ip-address", cloudkitPrefix)
	cloudkitAAPReconciledConfigVersionAnnotation     string = fmt.Sprintf("%s/reconciled-config-version", cloudkitPrefix)
)
