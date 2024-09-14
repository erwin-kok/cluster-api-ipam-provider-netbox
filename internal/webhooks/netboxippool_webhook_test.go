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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	ipamv1alpha1 "github.com/erwin-kok/cluster-api-ipam-provider-netbox/api/v1alpha1"
	"github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/index"
)

const ipamAPIVersion = "ipam.cluster.x-k8s.io/v1alpha1"

var _ = Describe("NetboxIPPool Webhook", func() {
	It("default webhook should work", func() {
		webhook := &NetboxIPPoolWebhook{}
		netboxIPPool := &ipamv1alpha1.NetboxIPPool{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"}}
		Expect(webhook.Default(ctx, netboxIPPool)).To(Succeed())
	})

	It("test creating NetboxIPPool", func() {
		scheme := runtime.NewScheme()
		Expect(ipamv1.AddToScheme(scheme)).To(Succeed())

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithIndex(&ipamv1.IPAddress{}, index.IPAddressPoolRefCombinedField, index.IPAddressByCombinedPoolRef).
			Build()

		webhook := NetboxIPPoolWebhook{
			Client: fakeClient,
		}

		namespacedPool := &ipamv1alpha1.NetboxIPPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-pool",
				Namespace: "test-namespace",
			},
			Spec: ipamv1alpha1.NetboxIPPoolSpec{
				CredentialsRef: &corev1.SecretReference{Name: "a-secret"},
				Type:           ipamv1alpha1.PrefixType,
				CIDR:           "192.168.1.0/24",
			},
		}

		_, err := webhook.ValidateCreate(ctx, namespacedPool)
		Expect(err).ToNot(HaveOccurred(), "should allow pool without Gateway")
	})

	It("test updating NetboxIPPool", func() {
		scheme := runtime.NewScheme()
		Expect(ipamv1.AddToScheme(scheme)).To(Succeed())

		namespacedPool := &ipamv1alpha1.NetboxIPPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-pool",
				Namespace: "test-namespace",
			},
			Spec: ipamv1alpha1.NetboxIPPoolSpec{
				CredentialsRef: &corev1.SecretReference{Name: "a-secret"},
				Type:           ipamv1alpha1.PrefixType,
				CIDR:           "192.168.1.0/24",
				Gateway:        "192.168.1.1",
			},
		}

		ips := []client.Object{
			createIP("address00", "192.168.1.2", namespacedPool),
			createIP("address01", "192.168.1.3", namespacedPool),
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(ips...).
			WithIndex(&ipamv1.IPAddress{}, index.IPAddressPoolRefCombinedField, index.IPAddressByCombinedPoolRef).
			Build()

		webhook := NetboxIPPoolWebhook{
			Client: fakeClient,
		}

		oldNamespacedPool := namespacedPool.DeepCopyObject()
		namespacedPool.Spec.CIDR = "192.168.2.0/24"
		namespacedPool.Spec.Gateway = "192.168.2.1"

		_, err := webhook.ValidateUpdate(ctx, oldNamespacedPool, namespacedPool)
		Expect(err).ToNot(HaveOccurred(), "should not allow removing in use IPs from addresses field in pool")
	})

	It("test deleting with existing ip addresses", func() {
		scheme := runtime.NewScheme()
		Expect(ipamv1.AddToScheme(scheme)).To(Succeed())

		namespacedPool := &ipamv1alpha1.NetboxIPPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-pool",
				Namespace: "test-namespace",
			},
			Spec: ipamv1alpha1.NetboxIPPoolSpec{
				CredentialsRef: &corev1.SecretReference{Name: "a-secret"},
				Type:           ipamv1alpha1.PrefixType,
				CIDR:           "192.168.1.0/24",
				Gateway:        "192.168.1.1",
			},
		}

		ips := []client.Object{
			createIP("address00", "192.168.1.2", namespacedPool),
			createIP("address01", "192.168.1.3", namespacedPool),
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(ips...).
			WithIndex(&ipamv1.IPAddress{}, index.IPAddressPoolRefCombinedField, index.IPAddressByCombinedPoolRef).
			Build()

		webhook := NetboxIPPoolWebhook{
			Client: fakeClient,
		}

		_, err := webhook.ValidateDelete(ctx, namespacedPool)
		Expect(err).To(HaveOccurred(), "should not allow deletion when claims exist")

		Expect(fakeClient.DeleteAllOf(ctx, &ipamv1.IPAddress{})).To(Succeed())

		_, err = webhook.ValidateDelete(ctx, namespacedPool)
		Expect(err).ToNot(HaveOccurred(), "should allow deletion when no claims exist")
	})

	It("test deleting with existing ip addresses and skip deletion", func() {
		scheme := runtime.NewScheme()
		Expect(ipamv1.AddToScheme(scheme)).To(Succeed())

		namespacedPool := &ipamv1alpha1.NetboxIPPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-pool",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					SkipValidateDeleteWebhookAnnotation: "",
				},
			},
			Spec: ipamv1alpha1.NetboxIPPoolSpec{
				CredentialsRef: &corev1.SecretReference{Name: "a-secret"},
				Type:           ipamv1alpha1.PrefixType,
				CIDR:           "192.168.1.0/24",
				Gateway:        "192.168.1.1",
			},
		}

		ips := []client.Object{
			createIP("address00", "192.168.1.2", namespacedPool),
			createIP("address01", "192.168.1.3", namespacedPool),
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(ips...).
			WithIndex(&ipamv1.IPAddress{}, index.IPAddressPoolRefCombinedField, index.IPAddressByCombinedPoolRef).
			Build()

		webhook := NetboxIPPoolWebhook{
			Client: fakeClient,
		}

		_, err := webhook.ValidateDelete(ctx, namespacedPool)
		Expect(err).ToNot(HaveOccurred(), "should not allow deletion when claims exist")
	})

	Context("test invalid scenarios", func() {
		DescribeTable("test invalid scenarios",
			func(spec ipamv1alpha1.NetboxIPPoolSpec, expectedError string) {
				namespacedPool := &ipamv1alpha1.NetboxIPPool{Spec: spec}

				scheme := runtime.NewScheme()
				Expect(ipamv1.AddToScheme(scheme)).To(Succeed())

				webhook := NetboxIPPoolWebhook{
					Client: fake.NewClientBuilder().
						WithScheme(scheme).
						WithIndex(&ipamv1.IPAddress{}, index.IPAddressPoolRefCombinedField, index.IPAddressByCombinedPoolRef).
						Build(),
				}

				By("create")
				Expect(testCreate(context.Background(), namespacedPool, webhook)).To(MatchError(ContainSubstring(expectedError)))

				By("update")
				Expect(testUpdate(context.Background(), namespacedPool, webhook)).To(MatchError(ContainSubstring(expectedError)))

				By("delete")
				Expect(testDelete(context.Background(), namespacedPool, webhook)).To(Succeed())
			},

			Entry("addresses must be set",
				ipamv1alpha1.NetboxIPPoolSpec{
					CIDR:           "",
					CredentialsRef: &corev1.SecretReference{Name: "a-secret"},
				},
				"CIDR is required",
			),

			Entry("CredentialsRef is required",
				ipamv1alpha1.NetboxIPPoolSpec{
					CIDR: "10.120.0.0/16",
				},
				"CredentialsRef is required",
			),

			Entry("CredentialsRef.Name is required",
				ipamv1alpha1.NetboxIPPoolSpec{
					CIDR:           "10.120.0.0/16",
					CredentialsRef: &corev1.SecretReference{},
				},
				"CredentialsRef.Name is required",
			),

			Entry("invalid subnet should not be allowed",
				ipamv1alpha1.NetboxIPPoolSpec{
					CIDR:           "10.0.0.3/34",
					Gateway:        "10.0.0.1",
					CredentialsRef: &corev1.SecretReference{Name: "a-secret"},
				},
				"CIDR is not a valid CIDR",
			),

			Entry("invalid gateway should not be allowed",
				ipamv1alpha1.NetboxIPPoolSpec{
					CIDR:           "10.0.0.3/16",
					Gateway:        "10.0.0.1.2",
					CredentialsRef: &corev1.SecretReference{Name: "a-secret"},
				},
				"Gateway is not a valid IP address",
			),

			Entry("invalid gateway should not be allowed",
				ipamv1alpha1.NetboxIPPoolSpec{
					CIDR:           "10.0.0.3/16",
					Gateway:        "10.0.0.300",
					CredentialsRef: &corev1.SecretReference{Name: "a-secret"},
				},
				"Gateway is not a valid IP address",
			),

			Entry("gateway must be within CIDR range",
				ipamv1alpha1.NetboxIPPoolSpec{
					CIDR:           "192.168.0.0/16",
					Gateway:        "10.0.0.1",
					CredentialsRef: &corev1.SecretReference{Name: "a-secret"},
				},
				"CIDR must contain gateway",
			),

			Entry("IPv4 subnet and IPv6 gateway should not be allowed",
				ipamv1alpha1.NetboxIPPoolSpec{
					CIDR:           "10.0.0.3/30",
					Gateway:        "2001:db8::1",
					CredentialsRef: &corev1.SecretReference{Name: "a-secret"},
				},
				"CIDR and gateway are mixed IPv4 and IPv6 addresses",
			),

			Entry("IPv4 subnet and IPv6 gateway should not be allowed",
				ipamv1alpha1.NetboxIPPoolSpec{
					CIDR:           "2001:db8::0/64",
					Gateway:        "10.0.0.1",
					CredentialsRef: &corev1.SecretReference{Name: "a-secret"},
				},
				"CIDR and gateway are mixed IPv4 and IPv6 addresses",
			),
		)
	})
})

func testCreate(ctx context.Context, obj runtime.Object, webhook NetboxIPPoolWebhook) error {
	createCopy := obj.DeepCopyObject()
	if err := webhook.Default(ctx, createCopy); err != nil {
		return err
	}
	_, err := webhook.ValidateCreate(ctx, createCopy)
	return err
}

func testDelete(ctx context.Context, obj runtime.Object, webhook NetboxIPPoolWebhook) error {
	deleteCopy := obj.DeepCopyObject()
	if err := webhook.Default(ctx, deleteCopy); err != nil {
		return err
	}
	_, err := webhook.ValidateDelete(ctx, deleteCopy)
	return err
}

func testUpdate(ctx context.Context, obj runtime.Object, webhook NetboxIPPoolWebhook) error {
	updateCopy := obj.DeepCopyObject()
	updatedCopy := obj.DeepCopyObject()
	err := webhook.Default(ctx, updateCopy)
	if err != nil {
		return err
	}
	err = webhook.Default(ctx, updatedCopy)
	if err != nil {
		return err
	}
	_, err = webhook.ValidateUpdate(ctx, updateCopy, updatedCopy)
	return err
}

func createIP(name string, ip string, pool *ipamv1alpha1.NetboxIPPool) *ipamv1.IPAddress {
	return &ipamv1.IPAddress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IPAddress",
			APIVersion: ipamAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: pool.Namespace,
		},
		Spec: ipamv1.IPAddressSpec{
			PoolRef: corev1.TypedLocalObjectReference{
				APIGroup: ptr.To[string](pool.GetObjectKind().GroupVersionKind().Group),
				Kind:     pool.GetObjectKind().GroupVersionKind().Kind,
				Name:     pool.GetName(),
			},
			Address: ip,
		},
	}
}
