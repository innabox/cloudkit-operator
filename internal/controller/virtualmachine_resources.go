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
	"context"
	"encoding/hex"
	"fmt"
	"hash/fnv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/innabox/cloudkit-operator/api/v1alpha1"
	ovnv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/userdefinednetwork/v1"
	"github.com/samber/lo"
)

func (r *VirtualMachineReconciler) newNamespace(ctx context.Context, instance *v1alpha1.VirtualMachine) (*appResource, error) {
	log := ctrllog.FromContext(ctx)

	var namespaceList corev1.NamespaceList
	var namespaceName string

	if err := r.List(ctx, &namespaceList, labelSelectorFromVirtualMachineInstance(instance)); err != nil {
		log.Error(err, "failed to list namespaces")
		return nil, err
	}

	if len(namespaceList.Items) > 1 {
		return nil, fmt.Errorf("found multiple matching namespaces for %s", instance.GetName())
	}

	if len(namespaceList.Items) == 0 {
		namespaceName = generateVirtualMachineNamespaceName(instance)
		if namespaceName == "" {
			return nil, fmt.Errorf("failed to generate namespace name")
		}
	} else {
		namespaceName = namespaceList.Items[0].GetName()
	}

	// Get tenant name from annotation
	tenantName, exists := instance.Annotations[cloudkitTenantAnnotation]
	if !exists || tenantName == "" {
		return nil, fmt.Errorf("tenant annotation '%s' not found or empty for virtual machine %s", cloudkitTenantAnnotation, instance.GetName())
	}

	labels := commonLabelsFromVirtualMachine(instance)
	labels["k8s.ovn.org/primary-user-defined-network"] = ""
	labels[cloudkitNetworkLabel] = GetNetworkName(instance.GetNamespace(), tenantName)

	annotations := map[string]string{
		cloudkitTenantAnnotation: tenantName,
	}

	namespace := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        namespaceName,
			Labels:      labels,
			Annotations: annotations,
		},
	}

	mutateFn := func() error {
		mergedLabels := lo.Assign(namespace.GetLabels(), labels)
		if !lo.ElementsMatch(lo.Entries(namespace.GetLabels()), lo.Entries(mergedLabels)) {
			namespace.SetLabels(mergedLabels)
		}
		mergedAnnotations := lo.Assign(namespace.GetAnnotations(), annotations)
		if !lo.ElementsMatch(lo.Entries(namespace.GetAnnotations()), lo.Entries(mergedAnnotations)) {
			namespace.SetAnnotations(mergedAnnotations)
		}
		return nil
	}

	return &appResource{
		namespace,
		mutateFn,
	}, nil
}

// NewCUDN creates a ClusterUserDefinedNetwork object for the given tenant name.
func (r *VirtualMachineReconciler) NewCUDN(ctx context.Context, instance *v1alpha1.VirtualMachine) (*appResource, error) {

	ns, err := r.findNamespace(ctx, instance)
	if err != nil || ns == nil {
		return nil, fmt.Errorf("failed to find namespace for virtual machine %s: %w", instance.GetName(), err)
	}

	// Extract tenant name from annotation
	tenantName, exists := instance.Annotations[cloudkitTenantAnnotation]
	if !exists || tenantName == "" {
		return nil, fmt.Errorf("tenant annotation '%s' not found or empty for virtual machine %s", cloudkitTenantAnnotation, instance.GetName())
	}

	// Get network name
	networkName := GetNetworkName(instance.GetNamespace(), tenantName)

	cudn := &ovnv1.ClusterUserDefinedNetwork{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "k8s.ovn.org/v1",
			Kind:       "ClusterUserDefinedNetwork",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: networkName,
		},
		Spec: ovnv1.ClusterUserDefinedNetworkSpec{
			NamespaceSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					cloudkitNetworkLabel: networkName,
				},
			},
			Network: ovnv1.NetworkSpec{
				Topology: ovnv1.NetworkTopologyLayer2,
				Layer2: &ovnv1.Layer2Config{
					Role: ovnv1.NetworkRolePrimary,
					IPAM: &ovnv1.IPAMConfig{
						Lifecycle: ovnv1.IPAMLifecyclePersistent,
					},
					Subnets: []ovnv1.CIDR{
						"10.200.0.0/16",
					},
				},
			},
		},
	}

	mutateFn := func() error {
		// Tie the lifecycle of CUDN to VM's namespace, so that when the VM is deleted, the CUDN is also deleted.
		// We cannot use the VM as the owner, because CUDN is a cluster-scoped resource.
		if err := controllerutil.SetOwnerReference(ns, cudn, r.Scheme); err != nil {
			return err
		}
		// Ensure labels and annotations are set
		mergedLabels := lo.Assign(cudn.GetLabels(), map[string]string{
			"app.kubernetes.io/name": cloudkitAppName,
		})
		if !lo.ElementsMatch(lo.Entries(cudn.GetLabels()), lo.Entries(mergedLabels)) {
			cudn.SetLabels(mergedLabels)
		}
		mergedAnnotations := lo.Assign(cudn.GetAnnotations(), map[string]string{
			cloudkitTenantAnnotation: tenantName,
		})
		if !lo.ElementsMatch(lo.Entries(cudn.GetAnnotations()), lo.Entries(mergedAnnotations)) {
			cudn.SetAnnotations(mergedAnnotations)
		}

		return nil
	}

	return &appResource{
		cudn,
		mutateFn,
	}, nil
}

// GetNetworkName computes an FNV-1a hash of the tenant name and returns it as "vm-net-<hash>".
// This ensures the name meets Kubernetes resource name requirements (RFC 1123 subdomain).
func GetNetworkName(vmNamespace, tenantName string) string {
	hasher := fnv.New64a()
	hasher.Write([]byte(tenantName))
	hashBytes := hasher.Sum(nil)
	hash := hex.EncodeToString(hashBytes)
	return fmt.Sprintf("%s-%s", vmNamespace, hash)
}

func commonLabelsFromVirtualMachine(instance *v1alpha1.VirtualMachine) map[string]string {
	key := client.ObjectKeyFromObject(instance)
	return map[string]string{
		"app.kubernetes.io/name":        cloudkitAppName,
		cloudkitVirtualMachineNameLabel: key.Name,
	}
}

func labelSelectorFromVirtualMachineInstance(instance *v1alpha1.VirtualMachine) client.MatchingLabels {
	return client.MatchingLabels{
		cloudkitVirtualMachineNameLabel: instance.GetName(),
	}
}
