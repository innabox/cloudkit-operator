package v1alpha1

func (ci *ComputeInstance) SetComputeInstanceReferenceNamespace(name string) {
	ci.EnsureComputeInstanceReference()
	ci.Status.ComputeInstanceReference.Namespace = name
}

func (ci *ComputeInstance) SetComputeInstanceReferenceKubeVirtComputeInstanceName(name string) {
	ci.EnsureComputeInstanceReference()
	ci.Status.ComputeInstanceReference.KubeVirtComputeInstanceName = name
}

func (ci *ComputeInstance) EnsureComputeInstanceReference() {
	if ci.Status.ComputeInstanceReference == nil {
		ci.Status.ComputeInstanceReference = &ComputeInstanceReferenceType{}
	}
}

func (ci *ComputeInstance) GetComputeInstanceReferenceNamespace() string {
	if ci.Status.ComputeInstanceReference == nil {
		return ""
	}
	return ci.Status.ComputeInstanceReference.Namespace
}

func (ci *ComputeInstance) GetComputeInstanceReferenceKubeVirtComputeInstanceName() string {
	if ci.Status.ComputeInstanceReference == nil {
		return ""
	}
	return ci.Status.ComputeInstanceReference.KubeVirtComputeInstanceName
}

func (ci *ComputeInstance) SetTenantReferenceName(name string) {
	ci.EnsureTenantReference()
	ci.Status.TenantReference.Name = name
}

func (ci *ComputeInstance) SetTenantReferenceNamespace(name string) {
	ci.EnsureTenantReference()
	ci.Status.TenantReference.Namespace = name
}

func (ci *ComputeInstance) EnsureTenantReference() {
	if ci.Status.TenantReference == nil {
		ci.Status.TenantReference = &TenantReferenceType{}
	}
}

func (ci *ComputeInstance) GetTenantReferenceName() string {
	if ci.Status.TenantReference == nil {
		return ""
	}
	return ci.Status.TenantReference.Name
}

func (ci *ComputeInstance) GetTenantReferenceNamespace() string {
	if ci.Status.TenantReference == nil {
		return ""
	}
	return ci.Status.TenantReference.Namespace
}
