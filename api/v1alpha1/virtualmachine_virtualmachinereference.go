package v1alpha1

func (vm *VirtualMachine) SetVirtualMachineReferenceNamespace(name string) {
	vm.EnsureVirtualMachineReference()
	vm.Status.VirtualMachineReference.Namespace = name
}

func (vm *VirtualMachine) SetVirtualMachineReferenceKubeVirtVirtalMachineName(name string) {
	vm.EnsureVirtualMachineReference()
	vm.Status.VirtualMachineReference.KubeVirtVirtalMachineName = name
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

func (vm *VirtualMachine) GetVirtualMachineReferenceKubeVirtVirtalMachineName() string {
	if vm.Status.VirtualMachineReference == nil {
		return ""
	}
	return vm.Status.VirtualMachineReference.KubeVirtVirtalMachineName
}
