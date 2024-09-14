package controller

import (
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api-ipam-provider-in-cluster/pkg/ipamutil"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	ipamv1alpha1 "github.com/erwin-kok/cluster-api-ipam-provider-netbox/api/v1alpha1"
	"github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/logger"
	"github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/netbox"
	ipampredicates "github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/predicates"
)

// NetboxProviderAdapter is used as middle layer for provider integration.
type NetboxProviderAdapter struct {
	NetboxServiceFactory func(url, apiToken string) (netbox.Client, error)
}

var _ ipamutil.ProviderAdapter = &NetboxProviderAdapter{}

// IPAddressClaimHandler reconciles a InClusterIPPool object.
type IPAddressClaimHandler struct {
	client.Client
	claim                *ipamv1.IPAddressClaim
	pool                 *ipamv1alpha1.NetboxIPPool
	netboxServiceFactory func(url, apiToken string) (netbox.Client, error)
}

var _ ipamutil.ClaimHandler = &IPAddressClaimHandler{}

func (a *NetboxProviderAdapter) SetupWithManager(_ context.Context, b *ctrl.Builder) error {
	b.
		For(&ipamv1.IPAddressClaim{}, builder.WithPredicates(
			ipampredicates.ClaimReferencesPoolKind(metav1.GroupKind{
				Group: ipamv1alpha1.GroupVersion.Group,
				Kind:  ipamv1alpha1.NetboxIPPoolKind,
			}),
		)).
		WithOptions(controller.Options{
			// To avoid race conditions when allocating IP Addresses, we explicitly set this to 1
			MaxConcurrentReconciles: 1,
		}).
		Owns(&ipamv1.IPAddress{}, builder.WithPredicates(
			ipampredicates.AddressReferencesPoolKind(metav1.GroupKind{
				Group: ipamv1alpha1.GroupVersion.Group,
				Kind:  ipamv1alpha1.NetboxIPPoolKind,
			}),
		))

	return nil
}

func (a *NetboxProviderAdapter) ClaimHandlerFor(cl client.Client, claim *ipamv1.IPAddressClaim) ipamutil.ClaimHandler {
	return &IPAddressClaimHandler{
		Client:               cl,
		claim:                claim,
		netboxServiceFactory: a.NetboxServiceFactory,
	}
}

// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxippools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxippools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxippools/finalizers,verbs=update
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddressclaims/status;ipaddresses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddressclaims/status;ipaddresses/finalizers,verbs=update
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddressclaims,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch

// FetchPool fetches the (Global)InClusterIPPool.
func (h *IPAddressClaimHandler) FetchPool(ctx context.Context) (client.Object, *ctrl.Result, error) {
	log := logger.FromContext(ctx)

	h.pool = &ipamv1alpha1.NetboxIPPool{}
	if err := h.Client.Get(ctx, types.NamespacedName{Namespace: h.claim.Namespace, Name: h.claim.Spec.PoolRef.Name}, h.pool); err != nil {
		return nil, nil, errors.Wrap(err, "failed to fetch pool")
	}

	if h.pool == nil {
		err := errors.New("pool not found")
		log.Error(err, "the referenced pool could not be found")
		return nil, nil, nil
	}

	return h.pool, nil, nil
}

// EnsureAddress ensures that the IPAddress contains a valid address.
func (h *IPAddressClaimHandler) EnsureAddress(ctx context.Context, address *ipamv1.IPAddress) (*ctrl.Result, error) {
	return nil, nil
}

func (h *IPAddressClaimHandler) ReleaseAddress(_ context.Context) (*ctrl.Result, error) {
	return nil, nil
}

// GetPool returns local pool.
func (h *IPAddressClaimHandler) GetPool() client.Object {
	return h.pool
}
