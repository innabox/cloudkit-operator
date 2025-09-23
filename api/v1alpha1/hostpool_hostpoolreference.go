package v1alpha1

func (hp *HostPool) SetHostPoolReferenceNamespace(name string) {
	hp.EnsureHostPoolReference()
	hp.Status.HostPoolReference.Namespace = name
}

func (hp *HostPool) SetHostPoolReferenceServiceAccountName(name string) {
	hp.EnsureHostPoolReference()
	hp.Status.HostPoolReference.ServiceAccountName = name
}

func (hp *HostPool) SetHostPoolReferenceRoleBindingName(name string) {
	hp.EnsureHostPoolReference()
	hp.Status.HostPoolReference.RoleBindingName = name
}

func (hp *HostPool) EnsureHostPoolReference() {
	if hp.Status.HostPoolReference == nil {
		hp.Status.HostPoolReference = &HostPoolReferenceType{}
	}
}

func (hp *HostPool) GetHostPoolReferenceNamespace() string {
	if hp.Status.HostPoolReference == nil {
		return ""
	}
	return hp.Status.HostPoolReference.Namespace
}

func (hp *HostPool) GetHostPoolReferenceServiceAccountName() string {
	if hp.Status.HostPoolReference == nil {
		return ""
	}
	return hp.Status.HostPoolReference.ServiceAccountName
}

func (hp *HostPool) GetHostPoolReferenceRoleBindingName() string {
	if hp.Status.HostPoolReference == nil {
		return ""
	}
	return hp.Status.HostPoolReference.RoleBindingName
}
