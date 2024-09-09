package ipam

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NewIPAddress creates a new ipamv1.IPAddress with references to a pool and claim.
func NewIPAddress(claim *ipamv1.IPAddressClaim, pool client.Object) ipamv1.IPAddress {
	poolGVK := pool.GetObjectKind().GroupVersionKind()

	return ipamv1.IPAddress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claim.Name,
			Namespace: claim.Namespace,
		},
		Spec: ipamv1.IPAddressSpec{
			ClaimRef: corev1.LocalObjectReference{
				Name: claim.Name,
			},
			PoolRef: corev1.TypedLocalObjectReference{
				APIGroup: &poolGVK.Group,
				Kind:     poolGVK.Kind,
				Name:     pool.GetName(),
			},
		},
	}
}

// ensureIPAddressOwnerReferences ensures that an IPAddress has the
// IPAddressClaim and IPPool as an OwnerReference.
func ensureIPAddressOwnerReferences(scheme *runtime.Scheme, address *ipamv1.IPAddress, claim *ipamv1.IPAddressClaim, pool client.Object) error {
	if err := controllerutil.SetControllerReference(claim, address, scheme); err != nil {
		if _, ok := err.(*controllerutil.AlreadyOwnedError); !ok {
			return errors.Wrap(err, "Failed to update address's claim owner reference")
		}
	}

	if err := controllerutil.SetOwnerReference(pool, address, scheme); err != nil {
		return errors.Wrap(err, "Failed to update address's pool owner reference")
	}

	var poolRefIdx int
	poolGVK := pool.GetObjectKind().GroupVersionKind()
	for i, ownerRef := range address.GetOwnerReferences() {
		if ownerRef.APIVersion == poolGVK.GroupVersion().String() &&
			ownerRef.Kind == poolGVK.Kind &&
			ownerRef.Name == pool.GetName() {
			poolRefIdx = i
		}
	}

	address.OwnerReferences[poolRefIdx].Controller = ptr.To(false)
	address.OwnerReferences[poolRefIdx].BlockOwnerDeletion = ptr.To(true)

	return nil
}
