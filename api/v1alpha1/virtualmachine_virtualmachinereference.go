package v1alpha1

func (vm *VirtualMachine) SetVirtualMachineReferenceNamespace(name string) {
	vm.EnsureVirtualMachineReference()
	vm.Status.VirtualMachineReference.Namespace = name
}

func (vm *VirtualMachine) SetVirtualMachineReferenceKubeVirtVirtualMachineName(name string) {
	vm.EnsureVirtualMachineReference()
	vm.Status.VirtualMachineReference.KubeVirtVirtualMachineName = name
}

func (vm *VirtualMachine) EnsureVirtualMachineReference() {
	if vm.Status.VirtualMachineReference == nil {
		vm.Status.VirtualMachineReference = &VirtualMachineReferenceType{}
	}
}

func (vm *VirtualMachine) GetVirtualMachineReferenceNamespace() string {
	if vm.Status.VirtualMachineReference == nil {
		return ""
	}
	return vm.Status.VirtualMachineReference.Namespace
}

func (vm *VirtualMachine) GetVirtualMachineReferenceKubeVirtVirtualMachineName() string {
	if vm.Status.VirtualMachineReference == nil {
		return ""
	}
	return vm.Status.VirtualMachineReference.KubeVirtVirtualMachineName
}

func (vm *VirtualMachine) SetTenantReferenceName(name string) {
	vm.EnsureTenantReference()
	vm.Status.TenantReference.Name = name
}

func (vm *VirtualMachine) SetTenantReferenceNamespace(name string) {
	vm.EnsureTenantReference()
	vm.Status.TenantReference.Namespace = name
}

func (vm *VirtualMachine) EnsureTenantReference() {
	if vm.Status.TenantReference == nil {
		vm.Status.TenantReference = &TenantReferenceType{}
	}
}

func (vm *VirtualMachine) GetTenantReferenceName() string {
	if vm.Status.TenantReference == nil {
		return ""
	}
	return vm.Status.TenantReference.Name
}

func (vm *VirtualMachine) GetTenantReferenceNamespace() string {
	if vm.Status.TenantReference == nil {
		return ""
	}
	return vm.Status.TenantReference.Namespace
}
