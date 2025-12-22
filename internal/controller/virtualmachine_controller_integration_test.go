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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cloudkitv1alpha1 "github.com/innabox/cloudkit-operator/api/v1alpha1"
)

var _ = Describe("VirtualMachine Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		const namespaceName = "default"
		const tenantName = "test-tenant"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: namespaceName,
		}
		virtualmachine := &cloudkitv1alpha1.VirtualMachine{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind VirtualMachine")
			err := k8sClient.Get(ctx, typeNamespacedName, virtualmachine)
			if err != nil && errors.IsNotFound(err) {
				resource := &cloudkitv1alpha1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: namespaceName,
						Annotations: map[string]string{
							cloudkitTenantAnnotation: tenantName,
						},
					},
					Spec: cloudkitv1alpha1.VirtualMachineSpec{
						TemplateID: "test_template",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			By("Cleanup the specific resource instance VirtualMachine")
			err := k8sClient.Get(ctx, typeNamespacedName, virtualmachine)
			Expect(err).NotTo(HaveOccurred())

			// Now delete the resource
			err = k8sClient.Delete(ctx, virtualmachine)
			if err != nil && !errors.IsNotFound(err) {
				Expect(err).NotTo(HaveOccurred())
			}

			By("Reconciling the deleted resource")
			Eventually(func() error {
				controllerReconciler := &VirtualMachineReconciler{
					Client: k8sClient,
					Scheme: k8sClient.Scheme(),
				}
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				return err
			}).Should(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &VirtualMachineReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that a tenant was created")
			expectedTenantName := getTenantObjectName(tenantName)
			tenant := &cloudkitv1alpha1.Tenant{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      expectedTenantName,
				Namespace: namespaceName,
			}, tenant)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying tenant has correct spec name")
			Expect(tenant.Spec.Name).To(Equal(tenantName))

			By("Verifying tenant has correct labels")
			Expect(tenant.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", cloudkitAppName))

			By("Verifying tenant has owner reference to VirtualMachine")
			Expect(tenant.OwnerReferences).NotTo(BeEmpty())
			Expect(tenant.OwnerReferences[0].Name).To(Equal(resourceName))

			// verify that finalizer is set
			By("Verifying the finalizer is set on the VirtualMachine resource")
			vm := &cloudkitv1alpha1.VirtualMachine{}
			err = k8sClient.Get(ctx, typeNamespacedName, vm)
			Expect(err).NotTo(HaveOccurred())
			Expect(vm.Finalizers).To(ContainElement(cloudkitVirtualMachineFinalizer))

			By("Verifying tenant reference is set on VirtualMachine status")
			Expect(vm.Status.TenantReference).NotTo(BeNil())
			Expect(vm.Status.TenantReference.Name).To(Equal(tenant.Name))
			Expect(vm.Status.TenantReference.Namespace).To(Equal(tenant.Namespace))
		})
	})

	Context("handleDesiredConfigVersion", func() {
		var reconciler *VirtualMachineReconciler
		ctx := context.Background()

		BeforeEach(func() {
			reconciler = &VirtualMachineReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
		})

		It("should compute and store a version of the spec", func() {
			vm := &cloudkitv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm-hash",
					Namespace: "default",
				},
				Spec: cloudkitv1alpha1.VirtualMachineSpec{
					TemplateID:         "template-1",
					TemplateParameters: `{"key": "value"}`,
				},
			}

			err := reconciler.handleDesiredConfigVersion(ctx, vm)
			Expect(err).NotTo(HaveOccurred())
			Expect(vm.Status.DesiredConfigVersion).NotTo(BeEmpty())
		})

		It("should be idempotent - same spec produces same version", func() {
			vm := &cloudkitv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm-idempotent",
					Namespace: "default",
				},
				Spec: cloudkitv1alpha1.VirtualMachineSpec{
					TemplateID:         "template-1",
					TemplateParameters: `{"key": "value"}`,
				},
			}

			// First call
			err := reconciler.handleDesiredConfigVersion(ctx, vm)
			Expect(err).NotTo(HaveOccurred())
			firstVersion := vm.Status.DesiredConfigVersion
			Expect(firstVersion).NotTo(BeEmpty())

			// Second call with same spec
			err = reconciler.handleDesiredConfigVersion(ctx, vm)
			Expect(err).NotTo(HaveOccurred())
			secondVersion := vm.Status.DesiredConfigVersion
			Expect(secondVersion).To(Equal(firstVersion))

			// Third call with same spec
			err = reconciler.handleDesiredConfigVersion(ctx, vm)
			Expect(err).NotTo(HaveOccurred())
			thirdVersion := vm.Status.DesiredConfigVersion
			Expect(thirdVersion).To(Equal(firstVersion))
		})

		It("should produce different versions for different specs", func() {
			vm1 := &cloudkitv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm-diff-1",
					Namespace: "default",
				},
				Spec: cloudkitv1alpha1.VirtualMachineSpec{
					TemplateID:         "template-1",
					TemplateParameters: `{"key": "value1"}`,
				},
			}

			vm2 := &cloudkitv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm-diff-2",
					Namespace: "default",
				},
				Spec: cloudkitv1alpha1.VirtualMachineSpec{
					TemplateID:         "template-1",
					TemplateParameters: `{"key": "value2"}`,
				},
			}

			err := reconciler.handleDesiredConfigVersion(ctx, vm1)
			Expect(err).NotTo(HaveOccurred())

			err = reconciler.handleDesiredConfigVersion(ctx, vm2)
			Expect(err).NotTo(HaveOccurred())

			Expect(vm1.Status.DesiredConfigVersion).NotTo(Equal(vm2.Status.DesiredConfigVersion))
		})

		It("should produce different versions for different template IDs", func() {
			vm1 := &cloudkitv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm-template-1",
					Namespace: "default",
				},
				Spec: cloudkitv1alpha1.VirtualMachineSpec{
					TemplateID: "template-1",
				},
			}

			vm2 := &cloudkitv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm-template-2",
					Namespace: "default",
				},
				Spec: cloudkitv1alpha1.VirtualMachineSpec{
					TemplateID: "template-2",
				},
			}

			err := reconciler.handleDesiredConfigVersion(ctx, vm1)
			Expect(err).NotTo(HaveOccurred())

			err = reconciler.handleDesiredConfigVersion(ctx, vm2)
			Expect(err).NotTo(HaveOccurred())

			Expect(vm1.Status.DesiredConfigVersion).NotTo(Equal(vm2.Status.DesiredConfigVersion))
		})

		It("should produce same version regardless of order of calls", func() {
			vm1 := &cloudkitv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm-order-1",
					Namespace: "default",
				},
				Spec: cloudkitv1alpha1.VirtualMachineSpec{
					TemplateID:         "template-1",
					TemplateParameters: `{"key": "value"}`,
				},
			}

			vm2 := &cloudkitv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm-order-2",
					Namespace: "default",
				},
				Spec: cloudkitv1alpha1.VirtualMachineSpec{
					TemplateID:         "template-1",
					TemplateParameters: `{"key": "value"}`,
				},
			}

			// Call on vm1 first
			err := reconciler.handleDesiredConfigVersion(ctx, vm1)
			Expect(err).NotTo(HaveOccurred())

			// Then call on vm2
			err = reconciler.handleDesiredConfigVersion(ctx, vm2)
			Expect(err).NotTo(HaveOccurred())

			// Versions should be identical
			Expect(vm1.Status.DesiredConfigVersion).To(Equal(vm2.Status.DesiredConfigVersion))
		})
	})

	Context("handleReconciledConfigVersion", func() {
		var reconciler *VirtualMachineReconciler
		ctx := context.Background()

		BeforeEach(func() {
			reconciler = &VirtualMachineReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
		})

		It("should copy annotation to status when annotation exists", func() {
			expectedVersion := "test-version-value-12345"
			vm := &cloudkitv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm-current",
					Namespace: "default",
					Annotations: map[string]string{
						cloudkitAAPReconciledConfigVersionAnnotation: expectedVersion,
					},
				},
				Spec: cloudkitv1alpha1.VirtualMachineSpec{
					TemplateID: "template-1",
				},
			}

			err := reconciler.handleReconciledConfigVersion(ctx, vm)
			Expect(err).NotTo(HaveOccurred())
			Expect(vm.Status.ReconciledConfigVersion).To(Equal(expectedVersion))
		})

		It("should clear status when annotation does not exist", func() {
			vm := &cloudkitv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-vm-no-annotation",
					Namespace:   "default",
					Annotations: map[string]string{},
				},
				Spec: cloudkitv1alpha1.VirtualMachineSpec{
					TemplateID: "template-1",
				},
				Status: cloudkitv1alpha1.VirtualMachineStatus{
					ReconciledConfigVersion: "some-old-version",
				},
			}

			err := reconciler.handleReconciledConfigVersion(ctx, vm)
			Expect(err).NotTo(HaveOccurred())
			Expect(vm.Status.ReconciledConfigVersion).To(BeEmpty())
		})

		It("should clear status when annotations map is nil", func() {
			vm := &cloudkitv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm-nil-annotations",
					Namespace: "default",
				},
				Spec: cloudkitv1alpha1.VirtualMachineSpec{
					TemplateID: "template-1",
				},
				Status: cloudkitv1alpha1.VirtualMachineStatus{
					ReconciledConfigVersion: "some-old-version",
				},
			}

			err := reconciler.handleReconciledConfigVersion(ctx, vm)
			Expect(err).NotTo(HaveOccurred())
			Expect(vm.Status.ReconciledConfigVersion).To(BeEmpty())
		})

		It("should update status when annotation value changes", func() {
			vm := &cloudkitv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm-update",
					Namespace: "default",
					Annotations: map[string]string{
						cloudkitAAPReconciledConfigVersionAnnotation: "version-1",
					},
				},
				Spec: cloudkitv1alpha1.VirtualMachineSpec{
					TemplateID: "template-1",
				},
			}

			// First call
			err := reconciler.handleReconciledConfigVersion(ctx, vm)
			Expect(err).NotTo(HaveOccurred())
			Expect(vm.Status.ReconciledConfigVersion).To(Equal("version-1"))

			// Update annotation
			vm.Annotations[cloudkitAAPReconciledConfigVersionAnnotation] = "version-2"

			// Second call
			err = reconciler.handleReconciledConfigVersion(ctx, vm)
			Expect(err).NotTo(HaveOccurred())
			Expect(vm.Status.ReconciledConfigVersion).To(Equal("version-2"))
		})
	})
})
