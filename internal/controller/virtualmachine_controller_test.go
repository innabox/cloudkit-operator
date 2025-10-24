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
	"log"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cloudkitv1alpha1 "github.com/innabox/cloudkit-operator/api/v1alpha1"
)

var _ = Describe("VirtualMachine Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		virtualmachine := &cloudkitv1alpha1.VirtualMachine{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind VirtualMachine")
			err := k8sClient.Get(ctx, typeNamespacedName, virtualmachine)
			if err != nil && errors.IsNotFound(err) {
				resource := &cloudkitv1alpha1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
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

			// envtest doesn't delete namespaces
			// https://book.kubebuilder.io/reference/envtest.html#namespace-usage-limitation
			By("Checking that a namespace is terminating")
			Eventually(func() corev1.NamespacePhase {
				var namespaceList corev1.NamespaceList
				err := k8sClient.List(ctx, &namespaceList, client.MatchingLabels{
					cloudkitVirtualMachineNameLabel: resourceName,
				})

				log.Println(namespaceList)
				Expect(err).NotTo(HaveOccurred())
				Expect(namespaceList.Items).To(HaveLen(1))
				return namespaceList.Items[0].Status.Phase
			}).Should(Equal(corev1.NamespaceTerminating))
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

			By("Checking that a namespace was created")
			var namespaceList corev1.NamespaceList
			err = k8sClient.List(ctx, &namespaceList, client.MatchingLabels{
				cloudkitVirtualMachineNameLabel: resourceName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(namespaceList.Items).To(HaveLen(1))

			namespace := namespaceList.Items[0]

			By("Verifying namespace has correct labels")
			Expect(namespace.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", cloudkitAppName))

			// verify that finalizer is set
			By("Verifying the finalizer is set on the VirtualMachine resource")
			vm := &cloudkitv1alpha1.VirtualMachine{}
			err = k8sClient.Get(ctx, typeNamespacedName, vm)
			Expect(err).NotTo(HaveOccurred())
			Expect(vm.Finalizers).To(ContainElement(cloudkitVirtualMachineFinalizer))
		})
	})
})
