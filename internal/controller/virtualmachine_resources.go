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
	tenantName, exists := instance.Annotations[cloudkitVirtualMachineTenantAnnotation]
	if !exists || tenantName == "" {
		return nil, fmt.Errorf("tenant annotation '%s' not found or empty for virtual machine %s", cloudkitVirtualMachineTenantAnnotation, instance.GetName())
	}

	labels := commonLabels()
	labels[cloudkitVirtualMachineNameLabel] = instance.GetName()
	labels[cloudkitVirtualMachineNetworkNameLabel] = getNetworkName(tenantName)
	labels["k8s.ovn.org/primary-user-defined-network"] = ""

	namespace := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespaceName,
			Labels: labels,
		},
	}

	mutateFn := func() error {
		mergedLabels := lo.Assign(namespace.GetLabels(), labels)
		if !lo.ElementsMatch(lo.Entries(namespace.GetLabels()), lo.Entries(mergedLabels)) {
			namespace.SetLabels(mergedLabels)
		}
		return nil
	}

	return &appResource{
		namespace,
		mutateFn,
	}, nil
}

func commonLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name": cloudkitAppName,
	}
}

func labelSelectorFromVirtualMachineInstance(instance *v1alpha1.VirtualMachine) client.MatchingLabels {
	return client.MatchingLabels{
		cloudkitVirtualMachineNameLabel: instance.GetName(),
	}
}

// getNetworkName computes an FNV-1a hash of the tenant name and returns it as "vm-net-<hash>".
// This ensures the name meets Kubernetes resource name requirements (RFC 1123 subdomain).
func getNetworkName(tenantName string) string {
	hasher := fnv.New64a()
	hasher.Write([]byte(tenantName))
	hashBytes := hasher.Sum(nil)
	hash := hex.EncodeToString(hashBytes)
	return fmt.Sprintf("vm-net-%s", hash)
}

func (r *VirtualMachineReconciler) newUDN(ctx context.Context, instance *v1alpha1.VirtualMachine) (*appResource, error) {
	// Get tenant name from annotation
	tenantName, exists := instance.Annotations[cloudkitVirtualMachineTenantAnnotation]
	if !exists || tenantName == "" {
		return nil, fmt.Errorf("tenant annotation '%s' not found or empty", cloudkitVirtualMachineTenantAnnotation)
	}

	// Get the network name for the tenant
	networkName := getNetworkName(tenantName)

	labels := commonLabels()
	annotations := map[string]string{
		cloudkitVirtualMachineTenantAnnotation: tenantName,
	}
	// Create the ClusterUserDefinedNetwork object
	udn := &ovnv1.ClusterUserDefinedNetwork{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "k8s.ovn.org/v1",
			Kind:       "ClusterUserDefinedNetwork",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        networkName,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: ovnv1.ClusterUserDefinedNetworkSpec{
			NamespaceSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					cloudkitVirtualMachineNetworkNameLabel: networkName,
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
		mergedLabels := lo.Assign(udn.GetLabels(), labels)
		if !lo.ElementsMatch(lo.Entries(udn.GetLabels()), lo.Entries(mergedLabels)) {
			udn.SetLabels(mergedLabels)
		}
		return nil
	}

	return &appResource{
		udn,
		mutateFn,
	}, nil
}

// cleanupCUDNIfLastNamespace checks if the given namespace is the last one using its network,
// and deletes the ClusterUserDefinedNetwork if so.
func (r *VirtualMachineReconciler) cleanupCUDNIfLastNamespace(ctx context.Context, ns *corev1.Namespace) error {
	log := ctrllog.FromContext(ctx)

	// Get the network name from the namespace labels
	networkName, exists := ns.Labels[cloudkitVirtualMachineNetworkNameLabel]
	if !exists || networkName == "" {
		// No network name label, nothing to clean up
		return nil
	}

	// Find all namespaces with the same network name
	var namespaceList corev1.NamespaceList
	if err := r.List(ctx, &namespaceList, client.MatchingLabels{
		cloudkitVirtualMachineNetworkNameLabel: networkName,
	}); err != nil {
		return fmt.Errorf("failed to list namespaces with network name %s: %w", networkName, err)
	}

	// Filter out namespaces that are already terminating
	activeNamespaces := []corev1.Namespace{}
	for _, n := range namespaceList.Items {
		// Exclude the current namespace and any terminating namespaces
		if n.GetName() != ns.GetName() && n.Status.Phase != corev1.NamespaceTerminating {
			activeNamespaces = append(activeNamespaces, n)
		}
	}

	// If this is no active namespace using the network, delete the CUDN
	if len(activeNamespaces) == 0 {
		log.Info("deleting ClusterUserDefinedNetwork", "networkName", networkName)

		cudn := &ovnv1.ClusterUserDefinedNetwork{
			ObjectMeta: metav1.ObjectMeta{
				Name: networkName,
			},
		}

		if err := r.Client.Delete(ctx, cudn); err != nil {
			if client.IgnoreNotFound(err) == nil {
				// CUDN already deleted, that's fine
				log.Info("ClusterUserDefinedNetwork already deleted", "networkName", networkName)
				return nil
			}
			return fmt.Errorf("failed to delete ClusterUserDefinedNetwork %s: %w", networkName, err)
		}

		log.Info("deleted ClusterUserDefinedNetwork", "networkName", networkName)
	}

	return nil
}
