package v1alpha1

func (hp *HostPool) SetHostPoolReferenceNamespace(name string) {
	hp.EnsureHostPoolReference()
	hp.Status.HostPoolReference.Namespace = name
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
