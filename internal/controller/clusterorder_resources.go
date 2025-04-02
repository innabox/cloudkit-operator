package controller

import (
	"context"
	"fmt"

	cloudkitv1alpha1 "github.com/innabox/cloudkit-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ClusterOrderReconciler) newNamespace(ctx context.Context, instance *cloudkitv1alpha1.ClusterOrder) (*appResource, error) {
	namespaceName := instance.GetClusterReferenceNamespace()
	if namespaceName == "" {
		namespaceName = generateNamespaceName(instance)
		if namespaceName == "" {
			return nil, fmt.Errorf("failed to generate namespace name")
		}
	}

	namespace := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespaceName,
			Labels: commonLabelsFromOrder(instance),
		},
	}

	instance.SetClusterReferenceNamespace(namespaceName)

	mutateFn := func() error {
		ensureCommonLabels(instance, namespace)
		return nil
	}

	return &appResource{
		namespace,
		mutateFn,
		true,
	}, nil
}

func (r *ClusterOrderReconciler) newServiceAccount(ctx context.Context, instance *cloudkitv1alpha1.ClusterOrder) (*appResource, error) {
	namespaceName := instance.GetClusterReferenceNamespace()
	if namespaceName == "" {
		return nil, fmt.Errorf("unable to retrieve required information from spec.clusterReference")
	}

	serviceAccountName := defaultServiceAccountName
	instance.SetClusterReferenceServiceAccountName(serviceAccountName)

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: namespaceName,
			Labels:    commonLabelsFromOrder(instance),
		},
	}

	mutateFn := func() error {
		ensureCommonLabels(instance, sa)
		return nil
	}

	return &appResource{
		sa,
		mutateFn,
		true,
	}, nil
}

func (r *ClusterOrderReconciler) newAdminRoleBinding(ctx context.Context, instance *cloudkitv1alpha1.ClusterOrder) (*appResource, error) {
	namespaceName := instance.GetClusterReferenceNamespace()
	serviceAccountName := instance.GetClusterReferenceServiceAccountName()
	if namespaceName == "" || serviceAccountName == "" {
		return nil, fmt.Errorf("unable to retrieve required information from spec.clusterReference")
	}

	roleBindingName := defaultRoleBindingName

	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      serviceAccountName,
			Namespace: namespaceName,
		},
	}

	roleref := rbacv1.RoleRef{
		APIGroup: rbacv1.GroupName,
		Kind:     "ClusterRole",
		Name:     "admin",
	}

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: namespaceName,
			Labels:    commonLabelsFromOrder(instance),
		},
	}

	instance.SetClusterReferenceRoleBindingName(roleBindingName)

	mutateFn := func() error {
		ensureCommonLabels(instance, roleBinding)
		roleBinding.Subjects = subjects
		roleBinding.RoleRef = roleref
		return nil
	}

	return &appResource{
		roleBinding,
		mutateFn,
		true,
	}, nil
}

func ensureCommonLabels(instance *cloudkitv1alpha1.ClusterOrder, obj client.Object) {
	requiredLabels := commonLabelsFromOrder(instance)
	objLabels := obj.GetLabels()
	if objLabels == nil {
		objLabels = make(map[string]string)
	}
	for k, v := range requiredLabels {
		objLabels[k] = v
	}
	obj.SetLabels(objLabels)
}

func commonLabelsFromOrder(instance *cloudkitv1alpha1.ClusterOrder) map[string]string {
	key := client.ObjectKeyFromObject(instance)
	return map[string]string{
		"app.kubernetes.io/name":           cloudkitAppName,
		cloudkitClusterOrderNameLabel:      key.Name,
		cloudkitClusterOrderNamespaceLabel: key.Namespace,
	}
}
