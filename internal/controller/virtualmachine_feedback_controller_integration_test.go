//go:build integration

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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cloudkitv1alpha1 "github.com/innabox/cloudkit-operator/api/v1alpha1"
	privatev1 "github.com/innabox/cloudkit-operator/internal/api/private/v1"
	sharedv1 "github.com/innabox/cloudkit-operator/internal/api/shared/v1"
)

// mockVirtualMachinesClient is a mock implementation of VirtualMachinesClient for testing.
type mockVirtualMachinesClient struct {
	getResponse    *privatev1.VirtualMachinesGetResponse
	getError       error
	updateResponse *privatev1.VirtualMachinesUpdateResponse
	updateError    error
	updateCalled   bool
	updateCount    int
	lastUpdate     *privatev1.VirtualMachine
}

func (m *mockVirtualMachinesClient) List(ctx context.Context, in *privatev1.VirtualMachinesListRequest, opts ...grpc.CallOption) (*privatev1.VirtualMachinesListResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockVirtualMachinesClient) Get(ctx context.Context, in *privatev1.VirtualMachinesGetRequest, opts ...grpc.CallOption) (*privatev1.VirtualMachinesGetResponse, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	return m.getResponse, nil
}

func (m *mockVirtualMachinesClient) Create(ctx context.Context, in *privatev1.VirtualMachinesCreateRequest, opts ...grpc.CallOption) (*privatev1.VirtualMachinesCreateResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockVirtualMachinesClient) Delete(ctx context.Context, in *privatev1.VirtualMachinesDeleteRequest, opts ...grpc.CallOption) (*privatev1.VirtualMachinesDeleteResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockVirtualMachinesClient) Update(ctx context.Context, in *privatev1.VirtualMachinesUpdateRequest, opts ...grpc.CallOption) (*privatev1.VirtualMachinesUpdateResponse, error) {
	m.updateCalled = true
	m.updateCount++
	m.lastUpdate = in.GetObject()
	if m.updateError != nil {
		return nil, m.updateError
	}
	return m.updateResponse, nil
}

var _ = Describe("VirtualMachineFeedbackReconciler", func() {
	const (
		resourceName     = "test-vm"
		vmNamespace      = "default"
		vmID             = "test-vm-id"
		virtualMachineNS = "cloudkit-vm-orders"
	)

	var (
		ctx                context.Context
		typeNamespacedName types.NamespacedName
		mockClient         *mockVirtualMachinesClient
		reconciler         *VirtualMachineFeedbackReconciler
	)

	BeforeEach(func() {
		ctx = context.Background()
		typeNamespacedName = types.NamespacedName{
			Name:      resourceName,
			Namespace: virtualMachineNS,
		}
		mockClient = &mockVirtualMachinesClient{}
		reconciler = &VirtualMachineFeedbackReconciler{
			hubClient:               k8sClient,
			virtualMachinesClient:   mockClient,
			virtualMachineNamespace: virtualMachineNS,
		}

		// Create the namespace if it doesn't exist
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: virtualMachineNS,
			},
		}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: virtualMachineNS}, namespace)
		if err != nil && apierrors.IsNotFound(err) {
			Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
		}
	})

	Context("When reconciling a resource that doesn't exist", func() {
		It("should return without error", func() {
			request := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "non-existent",
					Namespace: virtualMachineNS,
				},
			}
			result, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())
			Expect(mockClient.updateCalled).To(BeFalse())
		})
	})

	Context("When reconciling a resource without the VM ID label", func() {
		BeforeEach(func() {
			vm := &cloudkitv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: virtualMachineNS,
				},
				Spec: cloudkitv1alpha1.VirtualMachineSpec{
					TemplateID: "test_template",
				},
			}
			Expect(k8sClient.Create(ctx, vm)).To(Succeed())
		})

		AfterEach(func() {
			vm := &cloudkitv1alpha1.VirtualMachine{}
			err := k8sClient.Get(ctx, typeNamespacedName, vm)
			if err == nil {
				Expect(k8sClient.Delete(ctx, vm)).To(Succeed())
			}
		})

		It("should skip reconciliation", func() {
			request := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}
			result, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())
			Expect(mockClient.updateCalled).To(BeFalse())
		})
	})

	Context("When reconciling a resource that is being deleted", func() {
		BeforeEach(func() {
			vm := &cloudkitv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: virtualMachineNS,
					Labels: map[string]string{
						cloudkitVirtualMachineIDLabel: vmID,
					},
					Finalizers: []string{cloudkitVirtualMachineFinalizer},
				},
				Spec: cloudkitv1alpha1.VirtualMachineSpec{
					TemplateID: "test_template",
				},
			}
			Expect(k8sClient.Create(ctx, vm)).To(Succeed())
			Expect(k8sClient.Delete(ctx, vm)).To(Succeed())
			// Set a valid getResponse in case the reconciler tries to fetch (shouldn't happen but prevents panic)
			mockClient.getResponse = &privatev1.VirtualMachinesGetResponse{
				Object: &privatev1.VirtualMachine{
					Id:   vmID,
					Spec: &privatev1.VirtualMachineSpec{},
					Status: &privatev1.VirtualMachineStatus{
						State: privatev1.VirtualMachineState_VIRTUAL_MACHINE_STATE_UNSPECIFIED,
					},
				},
			}
		})

		AfterEach(func() {
			vm := &cloudkitv1alpha1.VirtualMachine{}
			err := k8sClient.Get(ctx, typeNamespacedName, vm)
			if err == nil {
				// Force delete by removing finalizers
				vm.Finalizers = nil
				Expect(k8sClient.Update(ctx, vm)).To(Succeed())
			}
		})

		It("should skip feedback reconciliation", func() {
			request := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}
			result, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())
			Expect(mockClient.updateCalled).To(BeFalse()) // Should not update when deleting
		})
	})

	Context("When reconciling a valid resource", func() {
		BeforeEach(func() {
			// Reset mock client state
			mockClient.updateCalled = false
			mockClient.lastUpdate = nil

			vm := &cloudkitv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: virtualMachineNS,
					Labels: map[string]string{
						cloudkitVirtualMachineIDLabel: vmID,
					},
				},
				Spec: cloudkitv1alpha1.VirtualMachineSpec{
					TemplateID: "test_template",
				},
			}
			Expect(k8sClient.Create(ctx, vm)).To(Succeed())
			// Update status separately since Status is a subresource - need to get fresh copy
			err := k8sClient.Get(ctx, typeNamespacedName, vm)
			Expect(err).NotTo(HaveOccurred())
			vm.Status.Phase = cloudkitv1alpha1.VirtualMachinePhaseReady
			vm.Status.Conditions = []metav1.Condition{
				{
					Type:               string(cloudkitv1alpha1.VirtualMachineConditionAccepted),
					Status:             metav1.ConditionFalse,
					Reason:             "Accepted",
					Message:            "VM is accepted",
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(cloudkitv1alpha1.VirtualMachineConditionProgressing),
					Status:             metav1.ConditionTrue,
					Reason:             "Progressing",
					Message:            "VM is progressing",
					LastTransitionTime: metav1.Now(),
				},
			}
			Expect(k8sClient.Status().Update(ctx, vm)).To(Succeed())

			// Setup mock response
			mockClient.getResponse = &privatev1.VirtualMachinesGetResponse{
				Object: &privatev1.VirtualMachine{
					Id:   vmID,
					Spec: &privatev1.VirtualMachineSpec{},
					Status: &privatev1.VirtualMachineStatus{
						State: privatev1.VirtualMachineState_VIRTUAL_MACHINE_STATE_UNSPECIFIED,
					},
				},
			}
			mockClient.updateResponse = &privatev1.VirtualMachinesUpdateResponse{}
		})

		AfterEach(func() {
			vm := &cloudkitv1alpha1.VirtualMachine{}
			err := k8sClient.Get(ctx, typeNamespacedName, vm)
			if err == nil {
				Expect(k8sClient.Delete(ctx, vm)).To(Succeed())
			}
		})

		It("should successfully sync conditions and phase", func() {
			request := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}
			result, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())
			Expect(mockClient.updateCalled).To(BeTrue())
			Expect(mockClient.lastUpdate).NotTo(BeNil())
			Expect(mockClient.lastUpdate.GetStatus().GetState()).To(Equal(privatev1.VirtualMachineState_VIRTUAL_MACHINE_STATE_READY))
		})

		It("should sync Progressing condition to Progressing condition", func() {
			request := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockClient.updateCalled).To(BeTrue())

			// Check that the condition was synced
			vm := mockClient.lastUpdate
			found := false
			for _, cond := range vm.GetStatus().GetConditions() {
				if cond.GetType() == privatev1.VirtualMachineConditionType_VIRTUAL_MACHINE_CONDITION_TYPE_PROGRESSING {
					Expect(cond.GetStatus()).To(Equal(sharedv1.ConditionStatus_CONDITION_STATUS_TRUE))
					Expect(cond.GetMessage()).To(Equal("VM is progressing"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("should sync Progressing phase", func() {
			vm := &cloudkitv1alpha1.VirtualMachine{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, vm)).To(Succeed())
			vm.Status.Phase = cloudkitv1alpha1.VirtualMachinePhaseProgressing
			Expect(k8sClient.Status().Update(ctx, vm)).To(Succeed())

			request := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockClient.updateCalled).To(BeTrue())
			Expect(mockClient.lastUpdate.GetStatus().GetState()).To(Equal(privatev1.VirtualMachineState_VIRTUAL_MACHINE_STATE_PROGRESSING))
		})

		It("should sync Failed phase", func() {
			vm := &cloudkitv1alpha1.VirtualMachine{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, vm)).To(Succeed())
			vm.Status.Phase = cloudkitv1alpha1.VirtualMachinePhaseFailed
			Expect(k8sClient.Status().Update(ctx, vm)).To(Succeed())

			request := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockClient.updateCalled).To(BeTrue())
			Expect(mockClient.lastUpdate.GetStatus().GetState()).To(Equal(privatev1.VirtualMachineState_VIRTUAL_MACHINE_STATE_FAILED))
		})

		It("should update only once when reconciliation is run twice with same data", func() {
			// Reset update count
			mockClient.updateCount = 0
			mockClient.updateCalled = false

			request := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}

			// First reconciliation - should trigger an update
			result, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())
			Expect(mockClient.updateCount).To(Equal(1))
			Expect(mockClient.updateCalled).To(BeTrue())

			// Second reconciliation with same data - should NOT trigger another update
			// because the VM object in the fulfillment service now matches what we're trying to sync
			// We need to update the mock's getResponse to reflect the state after the first update
			mockClient.getResponse = &privatev1.VirtualMachinesGetResponse{
				Object: mockClient.lastUpdate,
			}

			// Run reconciliation again
			result, err = reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())
			// Update count should still be 1, not 2, because only the timestamp changed, not the status
			Expect(mockClient.updateCount).To(Equal(1))
		})
	})
})
