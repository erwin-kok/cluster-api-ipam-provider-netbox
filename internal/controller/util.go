package controller

import (
	"k8s.io/apimachinery/pkg/types"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	ipamv1alpha1 "github.com/erwin-kok/cluster-api-ipam-provider-netbox/api/v1alpha1"
)

func ipAddressToNetboxIPPool(ipAddress *ipamv1.IPAddress, kind string) []reconcile.Request {
	if ipAddress.Spec.PoolRef.APIGroup != nil &&
		*ipAddress.Spec.PoolRef.APIGroup == ipamv1alpha1.GroupVersion.Group &&
		ipAddress.Spec.PoolRef.Kind == kind {
		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{
				Namespace: ipAddress.Namespace,
				Name:      ipAddress.Spec.PoolRef.Name,
			},
		}}
	}
	return nil
}
