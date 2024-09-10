package pool

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/index"
)

// ListAddressesInUse fetches all IPAddresses belonging to the specified pool.
// Note: requires `index.ipAddressByCombinedPoolRef` to be set up.
func ListAddressesInUse(ctx context.Context, c client.Client, namespace string, poolRef corev1.TypedLocalObjectReference) ([]ipamv1.IPAddress, error) {
	addresses := &ipamv1.IPAddressList{}
	err := c.List(ctx, addresses,
		client.MatchingFields{
			index.IPAddressPoolRefCombinedField: index.IPPoolRefValue(poolRef),
		},
		client.InNamespace(namespace),
	)
	var addr []ipamv1.IPAddress
	for _, a := range addresses.Items {
		gv, _ := schema.ParseGroupVersion(a.APIVersion)
		if gv.Group == "ipam.cluster.x-k8s.io" {
			addr = append(addr, a)
		}
	}
	return addr, err
}
