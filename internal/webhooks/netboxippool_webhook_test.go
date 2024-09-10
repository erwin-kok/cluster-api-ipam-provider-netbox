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

package webhooks

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	ipamv1alpha1 "github.com/erwin-kok/cluster-api-ipam-provider-netbox/api/v1alpha1"
	"github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/index"
)

var _ = Describe("NetboxIPRangeGlobalPool Webhook", func() {

	Context("Pool deletion with existing IPAddresses", func() {
		It("should not allow deletion when claims exist", func() {
			scheme := runtime.NewScheme()
			Expect(ipamv1.AddToScheme(scheme)).To(Succeed())

			namespacedPool := &ipamv1alpha1.NetboxIPPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-pool",
				},
				Spec: ipamv1alpha1.NetboxIPPoolSpec{
					Address: "20.0.0.0/14",
					Gateway: "10.0.0.1",
				},
			}

			ips := []client.Object{}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(ips...).
				WithIndex(&ipamv1.IPAddress{}, index.IPAddressPoolRefCombinedField, index.IPAddressByCombinedPoolRef).
				Build()

			webhook := NetboxIPPool{
				Client: fakeClient,
			}

			Expect(webhook.ValidateDelete(ctx, namespacedPool)).Error().NotTo(BeNil(), "should not allow deletion when claims exist")
		})
	})

	Context("When creating NetboxIPRangeGlobalPool under Validating Webhook", func() {
		It("Should deny if a required field is empty", func() {

			// TODO(user): Add your logic here

		})

		It("Should admit if all required fields are provided", func() {

			// TODO(user): Add your logic here

		})
	})

})
