package controller

import (
	"context"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	. "sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	ipamv1alpha1 "github.com/erwin-kok/cluster-api-ipam-provider-netbox/api/v1alpha1"
)

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

func newPool(generateName, namespace string, secret *corev1.Secret, gateway string, addresses []string, prefix int) *ipamv1alpha1.NetboxIPPool {
	poolSpec := ipamv1alpha1.NetboxIPPoolSpec{
		Type: ipamv1alpha1.PrefixType,
		// Prefix:    prefix,
		Gateway: gateway,
		// Addresses: addresses,
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
