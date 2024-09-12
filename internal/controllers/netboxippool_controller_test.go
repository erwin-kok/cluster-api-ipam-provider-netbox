/*
Copyright 2024.

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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	ipamv1alpha1 "github.com/erwin-kok/cluster-api-ipam-provider-netbox/api/v1alpha1"
	"github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/netbox"
	"github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/netbox/mock"
)

var _ = Describe("NetboxIPPool Controller", func() {
	var (
		namespace        string
		credentialSecret *corev1.Secret
	)

	BeforeEach(func() {
		namespace = createNamespace()
		credentialSecret = createCredentialsSecret(namespace)

	})

	Describe("Pool usage status", func() {
		const testPool = "test-pool"
		var (
			mockCtrl          *gomock.Controller
			netboxMock        *nbmock.MockClient
			createdClaimNames []string
			pool              *ipamv1alpha1.NetboxIPPool
		)

		BeforeEach(func() {
			createdClaimNames = nil
			mockCtrl = gomock.NewController(GinkgoT())
			netboxMock = nbmock.NewMockClient(mockCtrl)
			netboxIPPoolReconciler := NetboxIPPoolReconciler{
				Client: mgr.GetClient(),
				Scheme: mgr.GetScheme(),
				netboxServiceFactory: func(host, apiToken string) netbox.Client {
					return netboxMock
				},
			}
			Expect(netboxIPPoolReconciler.SetupWithManager(mgr)).To(Succeed())
		})

		AfterEach(func() {
			for _, name := range createdClaimNames {
				deleteClaim(name, namespace)
			}
			Expect(k8sClient.Delete(context.Background(), pool)).To(Succeed())
			mockCtrl.Finish()
		})

		DescribeTable("it shows the total, used, free ip addresses in the pool",
			func(prefix int, addresses []string, gateway string, expectedTotal, expectedUsed, expectedFree int) {
				gomock.InOrder(
					netboxMock.EXPECT().GetPrefix(gomock.Any(), gomock.Any(), gomock.Any()).Return(&netbox.NetboxIPPool{}, nil),
					netboxMock.EXPECT().GetPrefix(gomock.Any(), gomock.Any(), gomock.Any()).Return(&netbox.NetboxIPPool{}, nil),
				)

				pool = newPool(testPool, namespace, credentialSecret, gateway, addresses, prefix)
				Expect(k8sClient.Create(context.Background(), pool)).To(Succeed())

				Eventually(Object(pool)).
					WithTimeout(5 * time.Second).WithPolling(100 * time.Millisecond).Should(
					HaveField("Status.Addresses.Total", Equal(expectedTotal)))

				Expect(pool.Status.Addresses.Used).To(Equal(0))
				Expect(pool.Status.Addresses.Free).To(Equal(expectedTotal))

				for i := range expectedUsed {
					claim := newClaim(fmt.Sprintf("test%d", i), namespace, pool.GetName())
					Expect(k8sClient.Create(context.Background(), &claim)).To(Succeed())
					createdClaimNames = append(createdClaimNames, claim.Name)
				}

				Eventually(Object(pool)).
					WithTimeout(5 * time.Second).WithPolling(100 * time.Millisecond).Should(
					HaveField("Status.Addresses.Used", Equal(expectedUsed)))
				poolStatus := pool.Status
				Expect(poolStatus.Addresses.Total).To(Equal(expectedTotal))
				Expect(poolStatus.Addresses.Free).To(Equal(expectedFree))
			},

			Entry("When there is 1 claim and no gateway",
				24, []string{"10.0.0.10-10.0.0.20"}, "", 0, 0, 0),
		)

		DescribeTable("it shows the out of range ips if any")
	})

	Context("when the pool has IPAddresses", func() {

	})
})
