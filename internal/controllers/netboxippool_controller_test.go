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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/cluster-api-ipam-provider-in-cluster/pkg/ipamutil"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	ipamv1alpha1 "github.com/erwin-kok/cluster-api-ipam-provider-netbox/api/v1alpha1"
	"github.com/erwin-kok/cluster-api-ipam-provider-netbox/pkg/netbox"
	nbmock "github.com/erwin-kok/cluster-api-ipam-provider-netbox/pkg/netbox/mock"
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
			netboxFactory := func(url, apiToken string) (netbox.Client, error) {
				return netboxMock, nil
			}
			Expect(
				(&NetboxIPPoolReconciler{
					Client:               mgr.GetClient(),
					Scheme:               mgr.GetScheme(),
					netboxServiceFactory: netboxFactory,
				}).SetupWithManager(mgr)).To(Succeed())
			Expect(
				(&ipamutil.ClaimReconciler{
					Client: mgr.GetClient(),
					Scheme: mgr.GetScheme(),
					Adapter: &NetboxProviderAdapter{
						NetboxServiceFactory: netboxFactory,
					},
				}).SetupWithManager(ctx, mgr),
			).To(Succeed())
		})

		AfterEach(func() {
			for _, name := range createdClaimNames {
				deleteClaim(name, namespace)
			}
			Expect(k8sClient.Delete(context.Background(), pool)).To(Succeed())
			mockCtrl.Finish()
		})

		DescribeTable("it shows the total, used, free ip addresses in the pool",
			func(prefix int, address string, gateway string, expectedTotal, expectedUsed, expectedFree int) {
				gomock.InOrder(
					netboxMock.EXPECT().GetPrefix(gomock.Any(), gomock.Any(), gomock.Any()).Return(&netbox.NetboxIPPool{}, nil),
					netboxMock.EXPECT().GetPrefix(gomock.Any(), gomock.Any(), gomock.Any()).Return(&netbox.NetboxIPPool{}, nil),
				)

				pool = newPool(testPool, namespace, credentialSecret, gateway, address)
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

			Entry("When there is 1 claim and no gateway", 24, "10.0.0.10/16", "", 0, 0, 0),
		)

		DescribeTable("it shows the out of range ips if any")
	})

	Context("when the pool has IPAddresses", func() {

	})
})

func createNamespace() string {
	namespaceObj := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-ns-",
		},
	}
	ExpectWithOffset(1, k8sClient.Create(context.Background(), &namespaceObj)).To(Succeed())
	return namespaceObj.Name
}

func createCredentialsSecret(namespace string) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-secret-",
			Namespace:    namespace,
		},
		StringData: map[string]string{
			"bla": "bla",
		},
	}
	EventuallyWithOffset(1, k8sClient.Create).WithArguments(context.Background(), secret).Should(Succeed())
	return secret
}

func newClaim(name, namespace, poolName string) ipamv1.IPAddressClaim {
	return ipamv1.IPAddressClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: ipamv1.IPAddressClaimSpec{
			PoolRef: corev1.TypedLocalObjectReference{
				APIGroup: ptr.To("ipam.cluster.x-k8s.io"),
				Kind:     "NetboxIPPool",
				Name:     poolName,
			},
		},
	}
}

func deleteClaim(name, namespace string) {
	claim := ipamv1.IPAddressClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	ExpectWithOffset(1, k8sClient.Delete(context.Background(), &claim)).To(Succeed())
	EventuallyWithOffset(1, Get(&claim)).Should(Not(Succeed()))
}

func newPool(generateName, namespace string, secret *corev1.Secret, gateway string, cidr string) *ipamv1alpha1.NetboxIPPool {
	poolSpec := ipamv1alpha1.NetboxIPPoolSpec{
		Type:    ipamv1alpha1.PrefixType,
		CIDR:    cidr,
		Gateway: gateway,
		CredentialsRef: &corev1.SecretReference{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
	}
	return &ipamv1alpha1.NetboxIPPool{
		ObjectMeta: metav1.ObjectMeta{GenerateName: generateName, Namespace: namespace},
		Spec:       poolSpec,
	}
}
