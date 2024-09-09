package controller

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/seancfoley/ipaddress-go/ipaddr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	ipamv1alpha1 "github.com/erwin-kok/cluster-api-ipam-provider-netbox/api/v1alpha1"
	poolutil "github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/pool"
)

const (
	PoolFinalizer = "netboxippool.ipam.cluster.x-k8s.io"
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

func genericReconcile(ctx context.Context, c client.Client, pool ipamv1alpha1.GenericNetboxPool) (_ ctrl.Result, reterr error) {
	patchHelper, err := patch.NewHelper(pool, c)
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		if err := patchHelper.Patch(ctx, pool); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	poolTypeRef := corev1.TypedLocalObjectReference{
		APIGroup: ptr.To(ipamv1alpha1.GroupVersion.Group),
		Kind:     pool.GetObjectKind().GroupVersionKind().Kind,
		Name:     pool.GetName(),
	}

	addressesInUse, err := poolutil.ListAddressesInUse(ctx, c, pool.GetNamespace(), poolTypeRef)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to list addresses")
	}

	// Handle deleted pools
	if !pool.GetDeletionTimestamp().IsZero() {
		return reconcileDelete(ctx, pool, addressesInUse)
	}

	// If the Pool doesn't have our finalizer, add it.
	// Requeue immediately after adding finalizer to avoid the race condition between init and delete
	if !ctrlutil.ContainsFinalizer(pool, PoolFinalizer) {
		ctrlutil.AddFinalizer(pool, PoolFinalizer)
		return reconcile.Result{}, nil
	}

	// Handle non-deleted clusters
	return reconcileNormal(ctx, pool, addressesInUse)
}

func reconcileDelete(ctx context.Context, pool ipamv1alpha1.GenericNetboxPool, addressesInUse []ipamv1.IPAddress) (reconcile.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	if len(addressesInUse) > 0 {
		logger.Info(
			fmt.Sprintf("%d addresses are still in use", len(addressesInUse)),
			"Pool", klog.KObj(pool),
		)
		return ctrl.Result{}, nil
	}

	// Pool is deleted so remove the finalizer.
	ctrlutil.RemoveFinalizer(pool, PoolFinalizer)

	return reconcile.Result{}, nil
}

func reconcileNormal(ctx context.Context, pool ipamv1alpha1.GenericNetboxPool, addressesInUse []ipamv1.IPAddress) (reconcile.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	sequentialRange, err := pool.PoolSpec().ToSequentialRange()
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to build ip set from pool spec")
	}

	poolCount := (int)(sequentialRange.GetCount().Int64())
	if pool.PoolSpec().GetGateway() != "" {
		gatewayAddr, err := ipaddr.NewIPAddressString(pool.PoolSpec().GetGateway()).ToAddress()
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to parse pool gateway")
		}

		if sequentialRange.Contains(gatewayAddr) {
			poolCount--
		}
	}

	inUseCount := len(addressesInUse)
	free := poolCount - inUseCount

	pool.PoolStatus().SetAddresses(
		&ipamv1alpha1.NetboxPoolStatusIPAddresses{
			Total: poolCount,
			Used:  inUseCount,
			Free:  free,
		},
	)

	logger.Info("Updating pool with usage info", "statusAddresses", pool.PoolStatus().GetAddresses())

	return ctrl.Result{}, nil
}
