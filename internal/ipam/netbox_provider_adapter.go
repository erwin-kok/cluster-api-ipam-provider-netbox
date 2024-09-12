package ipam

import (
	"context"
	"fmt"
	"slices"

	"github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/index"
	"github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/logger"
	poolutil "github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/pool"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/annotations"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	ipamv1alpha1 "github.com/erwin-kok/cluster-api-ipam-provider-netbox/api/v1alpha1"
)

// NetboxProviderAdapter is used as middle layer for provider integration.
type NetboxProviderAdapter struct {
	Client           client.Client
	WatchFilterValue string
}

var _ ProviderAdapter = &NetboxProviderAdapter{}

// IPAddressClaimHandler reconciles a InClusterIPPool object.
type IPAddressClaimHandler struct {
	client.Client
	claim *ipamv1.IPAddressClaim
	pool  *ipamv1alpha1.NetboxIPPool
}

var _ ClaimHandler = &IPAddressClaimHandler{}

func (a *NetboxProviderAdapter) SetupWithManager(_ context.Context, b *ctrl.Builder) error {
	b.
		For(&ipamv1.IPAddressClaim{}, builder.WithPredicates(
			predicate.Or(
				ClaimReferencesPoolKind(metav1.GroupKind{
					Group: ipamv1alpha1.GroupVersion.Group,
					Kind:  ipamv1alpha1.NetboxIPPoolKind,
				}),
			),
		)).
		WithOptions(controller.Options{
			// To avoid race conditions when allocating IP Addresses, we explicitly set this to 1
			MaxConcurrentReconciles: 1,
		}).
		Watches(
			&ipamv1alpha1.NetboxIPPool{},
			handler.EnqueueRequestsFromMapFunc(a.netboxPoolToIPClaims()),
			builder.WithPredicates(predicate.Or(
				resourceTransitionedToUnpaused(),
				poolNoLongerEmpty(),
			)),
		)

	// TODO OWNS
	return nil
}

func (i *NetboxProviderAdapter) netboxPoolToIPClaims() func(context.Context, client.Object) []reconcile.Request {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		pool := obj.(*ipamv1alpha1.NetboxIPPool)
		requests := []reconcile.Request{}
		claims := &ipamv1.IPAddressClaimList{}
		err := i.Client.List(ctx, claims,
			client.MatchingFields{
				"index.poolRef": index.IPPoolRefValue(corev1.TypedLocalObjectReference{
					Name:     pool.GetName(),
					Kind:     ipamv1alpha1.NetboxIPPoolKind,
					APIGroup: &ipamv1alpha1.GroupVersion.Group,
				}),
			},
			client.InNamespace(pool.GetNamespace()),
		)
		if err != nil {
			return requests
		}
		for _, claim := range claims.Items {
			r := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      claim.Name,
					Namespace: claim.Namespace,
				},
			}
			requests = append(requests, r)
		}
		return requests
	}
}

func (a *NetboxProviderAdapter) ClaimHandlerFor(_ client.Client, claim *ipamv1.IPAddressClaim) ClaimHandler {
	return &IPAddressClaimHandler{
		Client: a.Client,
		claim:  claim,
	}
}

// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxippools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxippools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxippools/finalizers,verbs=update
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxglobalippools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxglobalippools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=netboxglobalippools/finalizers,verbs=update
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddressclaims,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddressclaims/status;ipaddresses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipam.cluster.x-k8s.io,resources=ipaddressclaims/status;ipaddresses/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch

// FetchPool fetches the (Global)InClusterIPPool.
func (h *IPAddressClaimHandler) FetchPool(ctx context.Context) (client.Object, *ctrl.Result, error) {
	log := logger.FromContext(ctx)

	if h.claim.Spec.PoolRef.Kind == ipamv1alpha1.NetboxIPPoolKind {
		icippool := &ipamv1alpha1.NetboxIPPool{}
		if err := h.Client.Get(ctx, types.NamespacedName{Namespace: h.claim.Namespace, Name: h.claim.Spec.PoolRef.Name}, icippool); err != nil {
			return nil, nil, errors.Wrap(err, "failed to fetch pool")
		}
		h.pool = icippool
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
	addressesInUse, err := poolutil.ListAddressesInUse(ctx, h.Client, h.pool.GetNamespace(), h.claim.Spec.PoolRef)
	if err != nil {
		return nil, fmt.Errorf("failed to list addresses: %w", err)
	}

	allocated := slices.ContainsFunc(addressesInUse, func(a ipamv1.IPAddress) bool {
		return a.Name == address.Name && a.Namespace == address.Namespace
	})

	if !allocated {

	}

	return nil, nil
}

func (h *IPAddressClaimHandler) ReleaseAddress(_ context.Context) (*ctrl.Result, error) {
	return nil, nil
}

func buildAddressList(addressesInUse []ipamv1.IPAddress, gateway string) []string {
	// Add extra capacity for the case that the pool's gateway is specified
	addrStrings := make([]string, len(addressesInUse), len(addressesInUse)+1)
	for i, address := range addressesInUse {
		addrStrings[i] = address.Spec.Address
	}

	if gateway != "" {
		addrStrings = append(addrStrings, gateway)
	}

	return addrStrings
}
func resourceTransitionedToUnpaused() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return annotations.HasPaused(e.ObjectOld) && !annotations.HasPaused(e.ObjectNew)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return !annotations.HasPaused(e.Object)
		},
	}
}

func poolStatus(o client.Object) *ipamv1alpha1.NetboxPoolStatusIPAddresses {
	pool, ok := o.(*ipamv1alpha1.NetboxIPPool)
	if !ok {
		return nil
	}
	return pool.Status.Addresses
}

// poolNoLongerEmpty only returns true if the Pool status previously had 0 free
// addresses and now has free addresses.
func poolNoLongerEmpty() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldStatus := poolStatus(e.ObjectOld)
			newStatus := poolStatus(e.ObjectNew)
			if oldStatus != nil && newStatus != nil {
				if oldStatus.Free == 0 && newStatus.Free > 0 {
					return true
				}
			}
			return false
		},
	}
}
